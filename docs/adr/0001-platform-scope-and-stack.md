# 1. Platform scope and stack

- Status: accepted
- Date: 2026-10-04

## Context

paved-road is a small internal developer platform that runs on a laptop. It
must show the day-to-day concerns of a platform team: self-service, golden
paths, infrastructure as code, observability, certificates and secrets. Anyone
should be able to start it with one command and no cloud account.

## Decision

| Concern | Choice |
|---|---|
| Workstation setup | Ansible playbook (`make bootstrap`) |
| Cluster | k3d (k3s in Docker), created by Terraform |
| Platform components | Terraform `helm_release` resources, pinned chart versions |
| Application delivery | Argo CD, app-of-apps, monorepo on GitHub |
| Ingress | Traefik shipped with k3s, `*.localhost` hostnames |
| Certificates | cert-manager with an internal root CA exposed as a ClusterIssuer |
| Metrics and dashboards | kube-prometheus-stack |
| Logs | Loki (single binary) fed by Grafana Alloy |
| Secrets in Git | Sealed Secrets (see [ADR 3](0003-sealed-secrets.md)) |
| Golden path tooling | `platformctl`, a Go CLI |

The platform layer (what every team relies on) is owned by Terraform. The
application layer (what teams create through the golden path) is owned by
Argo CD. This keeps a clear line between "the platform" and "the tenants".

## Consequences

- `make up` gives a working platform in about three minutes on a 16 GB laptop.
- `*.localhost` resolves to the loopback address without editing `/etc/hosts`.
- The platform layer is not reconciled continuously: drift is fixed by
  re-running `make platform`. Moving it under Argo CD is possible later, once
  the repository is reachable from the cluster.
