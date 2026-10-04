# Getting started

paved-road runs a complete internal developer platform on your machine: a
Kubernetes cluster (k3d), Argo CD, an internal certificate authority,
Prometheus, Grafana and Loki, Sealed Secrets, a PostgreSQL operator and, on
the full profile, a shared language model server.

## Requirements

| System | What to install first |
|---|---|
| macOS | Homebrew |
| Debian or Ubuntu | `sudo apt-get install -y make` |
| Windows | WSL2 with Ubuntu, then follow the Ubuntu path |

## First start

```bash
make bootstrap   # installs the toolchain with Ansible, starts Docker, picks a profile
make up          # creates the cluster and installs the platform with Terraform
make trust       # trusts the internal CA so browsers accept https://*.localhost
make creds       # prints the Argo CD and Grafana admin passwords
```

`make up` takes about three minutes. `make help` lists every target.

| Target | Effect |
|---|---|
| `make ask Q="..."`, `make chat` | ask the platform assistant one question, or several in a row |
| `make smoke` | lint and unit tests, as the CI runs on every push |
| `make regression` | end-to-end golden path test on the local cluster |
| `make down` | delete the cluster |
| `make clean` | delete the cluster and every generated file |
| `make fclean` | `clean`, plus caches, local images and the chosen profile |
| `make re` | `fclean`, `bootstrap` and `up`: rebuild from scratch |

`make fclean` keeps `~/.config/paved-road/` on purpose: it holds the Sealed
Secrets key (losing it makes sealed values in Git unreadable) and your tokens.

## Credentials for the private repository

Argo CD and the cluster nodes read the repository and the service images with
two read-only tokens. Export them before `make up`, or write them once in
`~/.config/paved-road/env` (one `KEY=value` per line, `chmod 600`):

- `PAVED_ROAD_GIT_TOKEN`: fine-grained token, Contents read-only, this repository only.
- `PAVED_ROAD_REGISTRY_TOKEN`: classic token with only `read:packages`, to pull images from GHCR.

`make cluster` and `make platform` refuse to run without them, because
applying without the Git token would remove Argo CD's access to the
repository. For a throwaway cluster, use `REQUIRE_TOKENS=0`.

## Resource profiles

| | full | lite |
|---|---|---|
| Recommended memory | 16 GB | 8 GB |
| Nodes | 2 | 1 |
| Logs (Loki) and Alertmanager | yes | no |
| Shared LLM server (Ollama) | yes | no |

`make bootstrap` picks the profile from the machine's memory and records it in
`.paved-road.env`. Force it with `make bootstrap PROFILE=lite`.

## Platform URLs

| UI | URL |
|---|---|
| Argo CD | https://argocd.localhost |
| Grafana | https://grafana.localhost |
| Any service | https://SERVICE.localhost |

`*.localhost` always resolves to your machine, no `/etc/hosts` edit needed.

## Checking the platform

```bash
make cli                 # builds bin/platformctl
bin/platformctl doctor   # nodes, platform deployments, internal CA, HTTPS
bin/platformctl status   # every service: Argo CD sync, health, ready pods, URL
```
