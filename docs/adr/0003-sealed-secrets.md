# 3. Sealed Secrets for secrets in Git

- Status: accepted
- Date: 2026-10-04

## Context

With GitOps, everything Argo CD deploys comes from Git, including the
configuration of secrets such as an LLM API key. Plain Kubernetes Secrets
cannot be committed.

Options considered:

1. **Sealed Secrets**: secrets are encrypted with the controller's public key
   and committed; only the in-cluster controller can decrypt them.
2. **External Secrets Operator + HashiCorp Vault**: Git only holds references,
   values live in Vault.

## Decision

Use Sealed Secrets on the local platform. It needs no external system and fits
the "one command, no account" goal.

## Consequences

- Developers seal secrets with `kubeseal`; the sealing key lives in
  `kube-system` and is regenerated with the cluster, so sealed files are tied
  to one cluster.
- In production, especially in a regulated environment, External Secrets with
  Vault (or the cloud provider's secret manager) is the better choice: central
  rotation, audit trail, short-lived credentials and no key material tied to a
  single cluster. The golden path only references a Secret by name, so this
  switch does not change application manifests.
