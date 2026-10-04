# Troubleshooting

Start with `bin/platformctl doctor`: it checks the nodes, the platform
deployments, the internal CA and HTTPS through the ingress.

## The browser rejects https://*.localhost

The internal CA is not trusted yet. Run `make trust` (it asks for sudo). Every
`make down && make up` creates a new CA, so run it again after recreating the
cluster. Firefox keeps its own trust store: import `.kube/paved-road-ca.crt`
there.

## A new service shows ImagePullBackOff

Right after the first push, `values.yaml` says `tag: latest` until the CI
commits the real tag: wait for the "Point Argo CD at the new images" job. If
it persists, check that `PAVED_ROAD_REGISTRY_TOKEN` has `read:packages` and
was set when the cluster was created (`make up`).

## Argo CD shows "authentication required"

The repository credentials are missing. Set `PAVED_ROAD_GIT_TOKEN` (or write
it in `~/.config/paved-road/env`) and run `make platform`.

## make up stops with "PAVED_ROAD_GIT_TOKEN ... must be set"

This guard prevents removing Argo CD's access to the private repository by
applying without credentials. Set both tokens, or use `REQUIRE_TOKENS=0` for
a throwaway cluster.

## A service does not appear in Argo CD

Only the `main` branch is deployed. Check that `apps/SERVICE/values.yaml` is
pushed, then look at the `services` ApplicationSet:

```bash
kubectl -n argocd get applicationset services -o yaml
```

## The service is not in Grafana

The dashboard is created by the chart as a ConfigMap labelled
`grafana_dashboard`. Metrics need `/metrics` to expose `http_requests_total`
and `http_request_duration_seconds`, which the templates already do. Check the
scrape target:

```bash
kubectl -n SERVICE get servicemonitor
```

## The cluster is slow or pods are evicted

Check memory with `kubectl top nodes`. On machines with less than 16 GB use
the lite profile: `make down`, then `make bootstrap PROFILE=lite && make up`.
On macOS, Colima's VM size is set by `make bootstrap` (8 GB on full, 4 GB on
lite).

## The assistant answers "Ollama request failed: Connection refused"

The shared model server restarted. Check it with
`kubectl -n ai get pods`; an `OOMKilled` last state means memory ran out. The
platform caps Ollama's prompt cache with `LLAMA_ARG_CACHE_RAM`
(`infra/platform/values/ollama.yaml`): if you changed the model or the memory
limit, keep the cache well below the limit. Ollama reloads the model on its
own; ask again after a minute.

## Logs

On the full profile, logs of every pod are in Grafana: Explore, data source
Loki, query `{namespace="SERVICE"}`. On the lite profile use
`kubectl -n SERVICE logs deploy/SERVICE`.
