# Secrets

Secrets live in Git, encrypted with Sealed Secrets. Only the cluster's
controller can decrypt them.

## Add a secret to a service

```bash
bin/platformctl seal rag-assistant ANTHROPIC_API_KEY
```

The command reads the value from your terminal without echoing it (or from
standard input when piped), encrypts it for this service only, and writes the
ciphertext under `secrets:` in `apps/rag-assistant/values.yaml`. Commit and
push: Argo CD creates the `rag-assistant-secrets` Secret and the service
receives `ANTHROPIC_API_KEY` as an environment variable.

Secrets are sealed with the "strict" scope: the ciphertext only decrypts for
that secret name in that namespace, so copying it to another service does not
work.

## The sealing key

The controller's key pair is kept outside the cluster in
`~/.config/paved-road/sealed-secrets/` and installed by `make platform`. This
is why sealed values still decrypt after `make down && make up`. The key is
created once by `make sealing-key`.

Keep this directory private and back it up: losing it means re-sealing every
secret. Sealed values are tied to this key, so a teammate with a different
key must re-seal them.

## Rotating a secret

Run `platformctl seal` again with the new value, commit and push. Argo CD
updates the Secret; restart the service to pick up the new environment:

```bash
kubectl -n SERVICE rollout restart deployment/SERVICE
```

## In production

A bank-grade setup would keep secret values in HashiCorp Vault or a cloud
secret manager, synced by External Secrets Operator: central rotation, audit
trail and short-lived credentials. Services would not change, since they only
read environment variables from a Secret by name. See ADR 3.
