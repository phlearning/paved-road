#!/usr/bin/env bash
# End-to-end test of the golden path, for every supported language:
# scaffold a service with platformctl, build its image, deploy it with the
# shared chart and check what the platform promises: HTTPS with the internal
# CA, Prometheus scraping and a Grafana dashboard.
#
# Argo CD is not involved: it needs the repository on GitHub, which a PR from
# a fork or a local checkout does not have. The chart and values it would
# deploy are exactly the ones used here.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CLUSTER="${CLUSTER_NAME:-paved-road}"
LANGUAGES=("${@:-python go}")
read -r -a LANGUAGES <<<"${LANGUAGES[*]}"
export KUBECONFIG="${KUBECONFIG:-$ROOT/.kube/config}"

PLATFORMCTL="$ROOT/bin/platformctl"
CA_FILE="$(mktemp)"
CREATED=()

log() { printf '\n\033[1m==> %s\033[0m\n' "$*"; }

cleanup() {
  for name in "${CREATED[@]}"; do
    helm uninstall "$name" -n "$name" --wait >/dev/null 2>&1 || true
    kubectl delete namespace "$name" --wait=false >/dev/null 2>&1 || true
    rm -rf "$ROOT/apps/$name"
  done
  rm -f "$CA_FILE"
}
trap cleanup EXIT

# Retries a command until it succeeds or the timeout (seconds) expires.
eventually() {
  local timeout="$1"; shift
  local deadline=$((SECONDS + timeout))
  until "$@" >/dev/null 2>&1; do
    if ((SECONDS >= deadline)); then
      echo "timed out after ${timeout}s: $*" >&2
      "$@" || true
      return 1
    fi
    sleep 3
  done
}

prometheus_target_up() {
  local job="$1"
  kubectl get --raw "/api/v1/namespaces/monitoring/services/kube-prometheus-stack-prometheus:9090/proxy/api/v1/query?query=up%7Bjob%3D%22${job}%22%7D" \
    | grep -q '"value":\[[0-9.]*,"1"\]'
}

log "Building platformctl"
(cd "$ROOT" && go build -o bin/platformctl ./cli/cmd/platformctl)

log "Checking the platform"
"$PLATFORMCTL" doctor
kubectl -n cert-manager get secret paved-road-root-ca -o jsonpath='{.data.ca\.crt}' | base64 -d >"$CA_FILE"

for lang in "${LANGUAGES[@]}"; do
  name="e2e-$lang"
  host="$name.localhost"
  CREATED+=("$name")

  log "[$lang] Scaffolding $name"
  (cd "$ROOT" && "$PLATFORMCTL" new-service "$name" --lang "$lang" --owner e2e)

  log "[$lang] Building and importing the image"
  image="$(sed -n 's/^  repository: //p' "$ROOT/apps/$name/values.yaml"):e2e"
  docker build -q -t "$image" "$ROOT/apps/$name"
  k3d image import "$image" -c "$CLUSTER"

  log "[$lang] Deploying with the shared chart"
  helm upgrade --install "$name" "$ROOT/platform/charts/service" \
    -n "$name" --create-namespace \
    -f "$ROOT/apps/$name/values.yaml" \
    --set-string image.tag=e2e --set image.pullPolicy=Never \
    --wait --timeout 3m

  log "[$lang] Checking HTTPS with the internal CA"
  eventually 120 curl -fsS --cacert "$CA_FILE" "https://$host/"
  curl -fsS --cacert "$CA_FILE" "https://$host/" && echo

  log "[$lang] Checking Prometheus scrapes the service"
  for _ in $(seq 5); do curl -fsS --cacert "$CA_FILE" "https://$host/" >/dev/null; done
  eventually 120 prometheus_target_up "$name"
  echo "target up{job=\"$name\"} = 1"

  log "[$lang] Checking the dashboard is published"
  kubectl -n "$name" get configmap "$name-dashboard" -o jsonpath='{.metadata.labels.grafana_dashboard}' | grep -qx 1
  echo "configmap $name-dashboard labelled for Grafana"
done

log "End-to-end test passed for: ${LANGUAGES[*]}"
