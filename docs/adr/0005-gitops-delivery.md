# 5. Golden path delivery with GitOps

- Status: accepted
- Date: 2026-10-04

## Context

A developer should go from "I need a service" to a running, observable, HTTPS
service without filing a ticket or writing Kubernetes manifests. The platform
team should still be able to change how every service is deployed in one
place, and the repository is private.

## Decision

**Scaffolding.** `platformctl new-service NAME --lang python|go` renders
`apps/NAME` from templates embedded in the binary (`templates/`): application
code with tests, a non-root Dockerfile, `service.yaml` (metadata) and
`values.yaml`. Both languages expose the same metrics
(`http_requests_total`, `http_request_duration_seconds`) so one dashboard
fits every service.

**One shared chart.** Services do not carry Kubernetes manifests. The chart in
`platform/charts/service` renders the Deployment (non-root, read-only root
filesystem, no capabilities), Service, Ingress with a certificate from the
internal CA, ServiceMonitor and Grafana dashboard. A JSON schema rejects
invalid values before anything reaches the cluster.

**Argo CD.** Terraform registers the repository and creates a root
Application that syncs `platform/argocd` (app of apps). It contains:

- an `AppProject` that limits services to their own namespaces and to a short
  list of resource kinds, so a service cannot touch the platform;
- an `ApplicationSet` with a Git directory generator: every `apps/*`
  directory becomes an Application that combines the shared chart with the
  service's `values.yaml` (Argo CD multiple sources).

**CI.** Two workflows. Smoke checks (`ci.yml`) run on every pull request and
push, in a few minutes: lint (Terraform, Ansible, shell, workflows), a Trivy
scan of the configuration and of the fully rendered shared chart, Go tests,
chart lint, then for each changed service its tests, an image build and a
Trivy scan that fails on fixable HIGH or CRITICAL vulnerabilities. On `main`,
the images are pushed to GHCR for amd64 and arm64 (laptops are often Apple
Silicon) and a bot commit sets the new tag in `values.yaml`; Argo CD deploys
that commit.

The regression suite (`regression.yml`) runs on demand (Actions tab or
`make regression-ci`): a fresh Ubuntu runner is bootstrapped with Ansible,
gets the platform in the lite profile, and `scripts/e2e.sh` scaffolds, builds
and deploys a service per language. It takes about 10 minutes, too slow for
every push; `make regression` runs the same test on the local cluster.

**Credentials.** Two read-only tokens, read from the environment by
Terraform and never committed:

| Variable | Scope | Used by |
|---|---|---|
| `PAVED_ROAD_GIT_TOKEN` | fine-grained, Contents: read, this repository only | Argo CD |
| `PAVED_ROAD_REGISTRY_TOKEN` | classic, `read:packages` only | k3s nodes pulling from GHCR |

Image pull credentials are configured at the node level (k3s
`registries.yaml`), so namespaces created by the ApplicationSet need no
`imagePullSecret`.

## Consequences

- Adding a service is a pull request; removing the directory removes the
  Application and its resources (automated prune).
- Upgrading every service (a new security context, a new label) is a change
  to one chart.
- The end-to-end test deploys with Helm directly: Argo CD would need access
  to the branch under test. The chart and values are the same ones Argo CD
  uses.
- GHCR does not accept fine-grained tokens, hence a classic token limited to
  `read:packages`. Changing the registry token recreates the k3d cluster.
- The repository URL appears in `platform/config.yaml` and in
  `platform/argocd/*.yaml`; renaming the repository means updating both.
