# 7. Portability to OpenShift

- Status: accepted
- Date: 2026-10-04

## Context

paved-road runs on k3s for a laptop, but enterprise platforms, including in
banking, often run Red Hat OpenShift. The golden path should not lock
services into k3s specifics, so that moving the platform means changing the
platform layer, not every service.

## Decision

Services are written to run unchanged under OpenShift's default
`restricted-v2` security context constraint:

| Requirement of restricted-v2 | What the golden path does |
|---|---|
| Arbitrary UID assigned by OpenShift | Pod specs never set `runAsUser`; images only need read access to their files (`chmod a+rX` on the baked model) and write to `/tmp`, an `emptyDir` |
| No privilege escalation, all capabilities dropped | `allowPrivilegeEscalation: false`, `capabilities.drop: [ALL]` |
| Seccomp profile | `seccompProfile: RuntimeDefault` |
| Non-root | `runAsNonRoot: true`, numeric `USER` in every Dockerfile |
| Read-only root filesystem (recommended) | `readOnlyRootFilesystem: true` |
| Unprivileged ports | Services listen on 8080 |

The platform layer maps to OpenShift components:

| paved-road on k3s | On OpenShift |
|---|---|
| Traefik + `Ingress` | OpenShift router; `Ingress` objects are converted to `Route`s automatically |
| cert-manager + internal CA | cert-manager Operator for Red Hat OpenShift, with the corporate CA as ClusterIssuer |
| Argo CD (Helm) | OpenShift GitOps operator (Argo CD), same ApplicationSet and AppProject |
| kube-prometheus-stack | User workload monitoring: `ServiceMonitor` objects work as is; dashboards move to a separately run Grafana |
| Loki + Alloy | OpenShift Logging with LokiStack |
| CloudNativePG | CloudNativePG or EDB Postgres for Kubernetes from OperatorHub; the `Cluster` resource is the same |
| Ollama | OpenShift AI model serving, or an internal model gateway |
| Sealed Secrets | External Secrets Operator with Vault (see ADR 3) |
| `infra/cluster` (k3d) | Cluster provisioned by the infrastructure team; `infra/cluster` is replaced, `infra/platform` mostly becomes operator subscriptions |

## Consequences

- The shared chart, the templates and the services need no change to run
  under `restricted-v2`. This is checked by Trivy on every pull request (no
  HIGH or CRITICAL misconfiguration in the rendered chart).
- Not tested on a real OpenShift cluster: OpenShift Local (CRC) needs about
  10 GB of memory on its own, beyond the laptop budget of this project.
- Grafana dashboards shipped as ConfigMaps depend on a Grafana sidecar, which
  OpenShift's built-in console does not have; the Grafana Operator's
  `GrafanaDashboard` resource would replace them.
