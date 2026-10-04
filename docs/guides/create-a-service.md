# Create a service

The golden path turns "I need a service" into a running, observable HTTPS
service with one command and one pull request. You never write Kubernetes
manifests.

## Create the service with platformctl new-service

```bash
make cli
bin/platformctl new-service orders-api --lang go   # or --lang python
```

Names use lowercase letters, digits and dashes, start with a letter and are at
most 40 characters long. The command creates `apps/orders-api/`:

| File | Purpose |
|---|---|
| `service.yaml` | Metadata read by the CI: name, language, owner, build settings |
| `values.yaml` | How the service is deployed (image, port, resources, capabilities) |
| `Dockerfile` | Non-root image; distroless for Go |
| application code and tests | `/`, `/healthz`, `/readyz` and `/metrics` already implemented |

## Deploy it: commit and push

```bash
git add apps/orders-api
git commit -m "Add orders-api"
git push
```

On every pull request the CI tests the service and builds its image. On
`main` it pushes the image to GHCR for amd64 and arm64, then commits the new
tag into `values.yaml`. Argo CD notices the commit within about three minutes
and deploys it.

Right after the first push the pod may show `ImagePullBackOff` for a minute:
`values.yaml` still says `tag: latest` until the CI commits the real tag.
This resolves itself.

## What you get

- `https://orders-api.localhost` with a certificate from the internal CA.
- A namespace named after the service, created by Argo CD.
- Prometheus scraping `/metrics` and a Grafana dashboard named
  "Service / orders-api" with request rate, error ratio, latency percentiles,
  available pods and memory.
- A hardened pod: non-root user, read-only root filesystem, no Linux
  capabilities, no service account token.

Check it with:

```bash
bin/platformctl status orders-api
```

## How it works

Every service is deployed by the same Helm chart, `platform/charts/service`,
owned by the platform team. A service only provides `values.yaml`, validated
by the chart's JSON schema. An Argo CD ApplicationSet creates one Application
per directory in `apps/`, so deleting the directory removes the service.

Services are limited by the `services` AppProject: they deploy into their own
namespace only and cannot create resources in `kube-system`, `argocd`,
`cert-manager` or `monitoring`.

## Shipping files from outside the service directory

By default the image is built from the service directory. A service that
needs other files from the repository (the RAG assistant ships `docs/`) sets
a build context and the paths that should trigger a rebuild in `service.yaml`:

```yaml
build:
  context: ../..        # relative to the service directory
  watch: [docs]         # relative to the repository root
```
