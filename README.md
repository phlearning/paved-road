# paved-road

🇫🇷 Français | [🇬🇧 English](README.en.md)

**paved-road** est une mini plateforme de développement interne (IDP) qui tourne sur un laptop. Elle reproduit le travail d'une équipe plateforme : un socle Kubernetes décrit en code, un *golden path* en self-service pour créer des services, de l'observabilité et des certificats intégrés par défaut, et des secrets gérés en GitOps.

Le premier service livré par ce golden path est un assistant RAG qui répond aux questions des développeurs à partir de la documentation de la plateforme.

## Architecture

```mermaid
flowchart LR
    dev([Développeur]) -->|platformctl new-service| repo[(Monorepo GitHub)]
    repo -->|GitOps| argocd[Argo CD]

    subgraph cluster[Cluster k3d]
        traefik[Traefik ingress] --> apps[Services applicatifs]
        argocd --> apps
        certmanager[cert-manager<br/>CA interne] -.->|certificats TLS| traefik
        sealed[Sealed Secrets] -.->|secrets| apps
        apps -->|métriques| prometheus[Prometheus]
        apps -->|logs| alloy[Alloy] --> loki[Loki]
        prometheus --> grafana[Grafana]
        loki --> grafana
    end

    ansible[Ansible] -->|prépare la machine| tf[Terraform]
    tf -->|crée le cluster et installe le socle| cluster
```

| Couche | Outils |
|---|---|
| Poste de travail | Ansible |
| Cluster | k3d (k3s), piloté par Terraform |
| Socle plateforme | Terraform + Helm, versions de charts épinglées |
| Déploiement | Argo CD (GitOps, app-of-apps) |
| Réseau et TLS | Traefik, cert-manager avec une CA interne |
| Observabilité | Prometheus, Grafana, Loki, Alloy |
| Secrets | Sealed Secrets |

Les choix sont expliqués dans les [ADR](docs/adr/).

## Démarrage rapide

| Système | Prérequis |
|---|---|
| macOS | [Homebrew](https://brew.sh) |
| Debian, Ubuntu | `sudo apt-get install -y make` |
| Windows | [WSL2](https://learn.microsoft.com/fr-fr/windows/wsl/install) avec Ubuntu, puis comme Ubuntu |

```bash
make bootstrap   # installe les outils, démarre Docker, choisit un profil (Ansible)
make up          # crée le cluster et installe la plateforme (Terraform)
make trust       # fait confiance à la CA interne (sudo)
make creds       # affiche les mots de passe admin
```

| Interface | URL |
|---|---|
| Argo CD | https://argocd.localhost |
| Grafana | https://grafana.localhost |

`make down` supprime le cluster, et `make help` liste toutes les commandes.

### Profils de ressources

| | `full` | `lite` |
|---|---|---|
| RAM conseillée | 16 Go | 8 Go |
| Nœuds | 2 | 1 |
| Logs (Loki) et Alertmanager | oui | non |

`make bootstrap` choisit le profil selon la RAM de la machine. Pour le forcer : `make bootstrap PROFILE=lite`. Détails dans l'[ADR 4](docs/adr/0004-resource-profiles-and-supported-systems.md).

## Structure du dépôt

```
ansible/          préparation du poste de travail (macOS, Debian/Ubuntu)
hack/             scripts utilitaires
infra/cluster/    cluster k3d (Terraform)
infra/platform/   socle plateforme (Terraform + Helm)
docs/adr/         décisions d'architecture
cli/              platformctl, le CLI du golden path (Go)
templates/        modèles de services
apps/             services déployés par Argo CD
```

## Feuille de route

- [x] **Socle** : Ansible (macOS, Linux, WSL2), profils full et lite, cluster k3d via Terraform, Argo CD, cert-manager et CA interne, Prometheus/Grafana/Loki, Sealed Secrets
- [ ] **Golden path** : CLI `platformctl` en Go, templates Python et Go, CI GitHub Actions, test e2e
- [ ] **Assistant RAG** : FastAPI, pgvector, Ollama ou API, `platformctl ask`
- [ ] **Finitions** : portabilité OpenShift, scan Trivy, vidéo de démo
