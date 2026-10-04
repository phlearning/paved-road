# hello

Service created with `platformctl new-service hello --lang go`.

| | |
|---|---|
| URL | https://hello.localhost |
| Dashboard | Grafana, "Service / hello" |
| Status | `platformctl status hello` |

Endpoints: `/` (hello), `/healthz`, `/readyz`, `/metrics`.

The deployment is described in `values.yaml` and rendered by the shared chart
in `platform/charts/service`: the image runs as non-root with a read-only
filesystem, gets a TLS certificate from the internal CA and is scraped by
Prometheus.
