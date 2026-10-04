# paved-road

[🇫🇷 Français](README.md) | 🇬🇧 English

**paved-road** is a small internal developer platform (IDP) that runs on a laptop. It mirrors the work of a platform team: a Kubernetes foundation described as code, a self-service *golden path* to create services, observability and certificates built in by default, and secrets managed the GitOps way.

The first service shipped through the golden path is a RAG assistant that answers developer questions from the platform's own documentation.

## Architecture

```mermaid
flowchart LR
    dev([Developer]) -->|platformctl new-service| repo[(GitHub monorepo)]
    repo -->|GitOps| argocd[Argo CD]

    subgraph cluster[k3d cluster]
        traefik[Traefik ingress] --> apps[Application services]
        argocd --> apps
        certmanager[cert-manager<br/>internal CA] -.->|TLS certificates| traefik
        sealed[Sealed Secrets] -.->|secrets| apps
        apps -->|metrics| prometheus[Prometheus]
        apps -->|logs| alloy[Alloy] --> loki[Loki]
        prometheus --> grafana[Grafana]
        loki --> grafana
    end

    ansible[Ansible] -->|prepares the workstation| tf[Terraform]
    tf -->|creates the cluster and installs the platform| cluster
```

| Layer | Tools |
|---|---|
| Workstation | Ansible |
| Cluster | k3d (k3s), driven by Terraform |
| Platform foundation | Terraform + Helm, pinned chart versions |
| Delivery | Argo CD (GitOps, app-of-apps) |
| Networking and TLS | Traefik, cert-manager with an internal CA |
| Observability | Prometheus, Grafana, Loki, Alloy |
| Secrets | Sealed Secrets |

Design decisions are recorded in the [ADRs](docs/adr/).

## Quick start

| System | Requirements |
|---|---|
| macOS | [Homebrew](https://brew.sh) |
| Debian, Ubuntu | `sudo apt-get install -y make` |
| Windows | [WSL2](https://learn.microsoft.com/en-us/windows/wsl/install) with Ubuntu, then as Ubuntu |

```bash
make bootstrap   # install the toolchain, start Docker, pick a profile (Ansible)
make up          # create the cluster and install the platform (Terraform)
make trust       # trust the internal CA (sudo)
make creds       # print the admin passwords
```

| UI | URL |
|---|---|
| Argo CD | https://argocd.localhost |
| Grafana | https://grafana.localhost |

`make down` deletes the cluster and `make help` lists every target.

### Resource profiles

| | `full` | `lite` |
|---|---|---|
| Recommended RAM | 16 GB | 8 GB |
| Nodes | 2 | 1 |
| Logs (Loki) and Alertmanager | yes | no |

`make bootstrap` picks the profile from the machine's memory. To force it: `make bootstrap PROFILE=lite`. See [ADR 4](docs/adr/0004-resource-profiles-and-supported-systems.md).

## Golden path: create a service

```bash
make cli                                          # build bin/platformctl
bin/platformctl new-service orders-api --lang go  # or --lang python
git add apps/orders-api && git commit -m "Add orders-api" && git push
bin/platformctl status orders-api
```

`new-service` generates the code, tests, a non-root Dockerfile and a `values.yaml`. Once merged, the CI builds the image and pushes it to GHCR, then Argo CD deploys the service to https://orders-api.localhost with a TLS certificate, Prometheus metrics and a Grafana dashboard. Every service shares the same Helm chart (`platform/charts/service`), owned by the platform team. See [ADR 5](docs/adr/0005-gitops-delivery.md).

`bin/platformctl doctor` checks the platform health, and `make e2e` runs the whole golden path locally for every language.

### Private repository access

Argo CD and the cluster nodes read the repository and the images with two read-only tokens, passed through the environment before `make up`:

```bash
export PAVED_ROAD_GIT_TOKEN=...       # fine-grained: Contents read, this repository only
export PAVED_ROAD_REGISTRY_TOKEN=...  # classic: read:packages only
```

## Repository layout

```
ansible/          workstation setup (macOS, Debian/Ubuntu)
hack/             helper scripts
infra/cluster/    k3d cluster (Terraform)
infra/platform/   platform foundation (Terraform + Helm)
docs/adr/         architecture decision records
platform/         shared service chart, Argo CD config, config.yaml
cli/              platformctl, the golden path CLI (Go)
templates/        service templates (Python, Go)
apps/             services deployed by Argo CD
.github/          CI: lint, tests, image builds, e2e
```

## Roadmap

- [x] **Foundation**: Ansible (macOS, Linux, WSL2), full and lite profiles, k3d cluster via Terraform, Argo CD, cert-manager and internal CA, Prometheus/Grafana/Loki, Sealed Secrets
- [x] **Golden path**: `platformctl` Go CLI, Python and Go templates, GitHub Actions CI, e2e test
- [ ] **RAG assistant**: FastAPI, pgvector, Ollama or API, `platformctl ask`
- [ ] **Polish**: OpenShift portability, Trivy scan, demo video
