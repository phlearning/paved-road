# 4. Resource profiles and supported systems

- Status: accepted
- Date: 2026-10-04

## Context

The first version targeted macOS with 16 GB of RAM. Measured on that setup,
the full platform uses about 3.7 GB inside the cluster, Grafana (about 390 MB)
and Prometheus (about 290 MB) being the largest workloads, plus the Colima VM
overhead. Developers on Linux, Windows or 8 GB laptops should be able to run
it too, and the CI runs on Ubuntu.

## Decision

**Supported systems**

| System | How |
|---|---|
| macOS | Homebrew and Colima |
| Debian, Ubuntu | apt (Docker, Terraform) and upstream binaries pinned by SHA-256 (k3d, kubectl, helm, kubeseal, Go) |
| Windows | WSL2 with Ubuntu, which follows the Linux path |

The Ansible playbook keeps one entry point and branches per OS family
(`ansible/tasks/darwin.yml`, `ansible/tasks/debian.yml`). On Linux, Docker
runs natively, which uses less memory than a VM on macOS. On WSL2 an existing
Docker Desktop installation is reused when present.

**Resource profiles**

| | full | lite |
|---|---|---|
| Nodes | server + agent | single server |
| Metrics and dashboards | yes | yes |
| Logs (Loki, Alloy) | yes | no |
| Alertmanager | yes | no |
| Argo CD notifications | yes | no |
| Prometheus retention | 2 days | 12 hours |

`make bootstrap` picks `full` from 15 GB of RAM and `lite` below, and records
the choice in `.paved-road.env`. `PROFILE=full|lite` overrides it.

## Consequences

- One code path per OS family keeps the playbook readable; adding Fedora means
  adding `tasks/redhat.yml`.
- Pinned digests make every binary download verifiable, at the cost of
  updating `ansible/vars/toolchain.yml` on every version bump.
- The lite profile has no central logs: `kubectl logs` is the fallback.
- The RAG assistant runs its LLM through an API in the lite profile, since a
  local model needs 2 to 3 GB more.
- Windows is supported through WSL2 only and is not tested in CI.
