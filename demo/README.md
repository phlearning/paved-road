# Démo

Deux supports pour présenter paved-road : un GIF du terminal généré
automatiquement, et le déroulé d'une vidéo d'environ 5 minutes.

## GIF du terminal

Sur une plateforme démarrée (`make up`) :

```bash
make cli
vhs demo/platformctl.tape
```

[VHS](https://github.com/charmbracelet/vhs) tape les commandes de
`platformctl.tape` dans un vrai terminal et produit `demo/platformctl.gif` :
`doctor`, `status`, `new-service` puis `ask`. Le service créé pour la démo est
supprimé à la fin.

## Vidéo (environ 5 minutes)

Préparation : `make up`, `make trust`, `make creds`, puis ouvrir Argo CD et
Grafana dans le navigateur. Poser une question à l'assistant une première fois
avant d'enregistrer, pour que le modèle soit déjà chargé en mémoire.

| Temps | Ce qu'on montre | Ce qu'on dit |
|---|---|---|
| 0:00 | Le README, le schéma d'architecture | Le problème : une équipe plateforme doit permettre aux développeurs de livrer seuls, de façon sûre et observable. paved-road est une plateforme interne complète qui tourne sur un laptop. |
| 0:30 | `make bootstrap` puis `make up` (accéléré) | Tout est du code : Ansible prépare la machine (macOS, Linux, WSL2), Terraform crée le cluster et installe le socle. Deux profils selon la RAM. |
| 1:15 | `bin/platformctl doctor` et `status` | Un CLI en Go pour les développeurs : santé de la plateforme, état de chaque service. |
| 1:45 | `platformctl new-service payments-api --lang go`, le contenu du dossier | Le golden path : code, tests, Dockerfile non-root, `values.yaml`. Aucun manifest Kubernetes à écrire. |
| 2:15 | GitHub Actions sur un push, puis Argo CD (ApplicationSet, apps synchronisées) | La CI teste, scanne avec Trivy, construit les images amd64 et arm64 et commite le tag. Argo CD déploie depuis Git. |
| 3:00 | https://hello.localhost, puis le dashboard Grafana « Service / hello » | Chaque service a son certificat de la CA interne, ses métriques et son dashboard, sans configuration. |
| 3:30 | `apps/rag-assistant/values.yaml`, puis `platformctl ask` en français | Les capacités en self-service : une base PostgreSQL avec pgvector, un Job d'indexation, le LLM partagé. L'assistant RAG est un service comme les autres. |
| 4:15 | Les logs du Job d'indexation (recall@5), le dashboard du RAG | La qualité de la recherche est mesurée à chaque déploiement, et les métriques (latence, tokens) sont dans Grafana. |
| 4:40 | `docs/adr/` (en particulier l'ADR 7 OpenShift) | Chaque choix est documenté : Sealed Secrets plutôt que Vault en local, portabilité vers OpenShift. |
