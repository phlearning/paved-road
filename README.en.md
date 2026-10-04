# paved-road

[🇫🇷 Français](README.md) | 🇬🇧 English

**paved-road** is a small internal developer platform (IDP) that runs on a laptop. It mirrors the work of a platform team: a Kubernetes foundation described as code, a self-service *golden path* to create services, observability and certificates built in by default, and secrets managed the GitOps way.

The first service shipped through the golden path is a RAG assistant that answers developer questions from the platform's own documentation.

![Terminal demo: doctor, status, new-service and the assistant](demo/platformctl.gif)

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
| Secrets | Sealed Secrets, `platformctl seal` |
| Databases | Self-service CloudNativePG (PostgreSQL 18, pgvector) |
| AI | Shared Ollama (`qwen3:1.7b`), or Claude through the API |
| Supply chain security | Trivy (configuration, rendered chart, images), multi-arch amd64/arm64 images |

Design decisions are recorded in the [ADRs](docs/adr/), the [guides](docs/guides/) cover day-to-day use, and the [lessons learned](docs/lessons-learned.md) (in French) describe the incidents met and how they were fixed.

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

| Command | Effect |
|---|---|
| `make ask Q="..."` / `make chat` | ask the platform assistant one question / several in a row |
| `make smoke` | fast checks (lint, unit tests), as the CI runs on every push |
| `make regression` | end-to-end golden path test on the local cluster (`make regression-ci` runs it on GitHub) |
| `make down` | delete the cluster |
| `make clean` / `make fclean` | delete the cluster and generated files / back to a fresh clone |
| `make re` | `fclean`, then `bootstrap` and `up`: rebuild everything from scratch |

`make help` lists every target.

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

### Self-service capabilities

A service turns them on in its `values.yaml`, without writing Kubernetes manifests:

| Capability | Key | What the platform does |
|---|---|---|
| PostgreSQL database | `postgres.enabled` | the operator creates the database, credentials arrive in `DATABASE_URL` |
| Post-deployment jobs | `jobs` | migrations, indexing... after every successful deployment |
| Secrets | `secrets` | values encrypted by `platformctl seal`, injected as environment variables |
| Dashboard | `dashboard.panels` | service-specific panels added to the standard dashboard |
| Shared LLM | `http://ollama.ai.svc:11434` | common model server (full profile) |

See the [platform-capabilities](docs/guides/platform-capabilities.md) guide.

### Private repository access

Argo CD and the cluster nodes read the repository and the images with two read-only tokens, passed through the environment before `make up`:

```bash
export PAVED_ROAD_GIT_TOKEN=...       # fine-grained: Contents read, this repository only
export PAVED_ROAD_REGISTRY_TOKEN=...  # classic: read:packages only
```

## Platform assistant

[`rag-assistant`](apps/rag-assistant/) is the first service created with the golden path. It answers developer questions from `docs/`, in the language of the question, citing its sources:

```bash
bin/platformctl ask "How do I add a PostgreSQL database to my service?"
```

It is a RAG: multilingual embeddings in-process, pgvector on the database provided by the platform, indexing by a Job after each deployment, and the shared LLM. Retrieval quality is measured (recall@5 of 100% on 12 reference questions, recomputed after every indexing). Design and measurements in [ADR 6](docs/adr/0006-rag-assistant.md).

## Repository layout

```
ansible/          workstation setup (macOS, Debian/Ubuntu)
scripts/          helper scripts
infra/cluster/    k3d cluster (Terraform)
infra/platform/   platform foundation (Terraform + Helm)
docs/adr/         architecture decision records
platform/         shared service chart, Argo CD config, config.yaml
cli/              platformctl, the golden path CLI (Go)
templates/        service templates (Python, Go)
apps/             services deployed by Argo CD
.github/          CI: smoke on every push, regression on demand
demo/             terminal GIF (VHS) and video script
```

## Roadmap

- [x] **Foundation**: Ansible (macOS, Linux, WSL2), full and lite profiles, k3d cluster via Terraform, Argo CD, cert-manager and internal CA, Prometheus/Grafana/Loki, Sealed Secrets
- [x] **Golden path**: `platformctl` Go CLI, Python and Go templates, GitHub Actions CI, e2e test
- [x] **RAG assistant**: FastAPI, pgvector, Ollama or API, `platformctl ask`, retrieval evaluation
- [x] **Security and portability**: Trivy scans in CI, multi-arch images, [OpenShift portability](docs/adr/0007-openshift-portability.md)
- [x] **Demo**: [terminal GIF and video script](demo/)
