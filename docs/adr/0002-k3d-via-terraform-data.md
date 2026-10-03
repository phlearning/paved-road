# 2. Drive k3d from Terraform with a config file

- Status: accepted
- Date: 2026-10-04

## Context

The cluster should be described as code and created by Terraform, like the
rest of the platform. The community Terraform providers for k3d
(`pvotal-tech/k3d`, `nikhilsbhat/k3d`) have not been released since 2023 and
do not follow current k3d versions.

## Decision

The cluster is described by a declarative k3d config file
(`infra/cluster/k3d.yaml.tftpl`). Terraform renders it and a `terraform_data`
resource reconciles it with the k3d CLI:

- the resource is replaced whenever the rendered config changes;
- creation is idempotent (`k3d cluster get` before `create`);
- a destroy-time provisioner deletes the cluster.

The platform components live in a second root module (`infra/platform`), so
the Helm and Kubernetes providers are only configured once the kubeconfig
exists.

## Consequences

- No dependency on an unmaintained provider; k3d upgrades are a version bump.
- Terraform only tracks the config, not the live cluster: a cluster deleted by
  hand is recreated on the next apply, but finer drift is not detected.
- The same approach maps to a managed cluster later: only `infra/cluster`
  changes (for example an `aws_eks_cluster`), `infra/platform` stays.
