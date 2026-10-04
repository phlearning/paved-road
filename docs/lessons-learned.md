# Retours d'expérience : incidents et solutions

Les problèmes réels rencontrés en construisant paved-road, avec leur cause et
ce qui a été mis en place. Chaque entrée suit le même format : symptôme,
cause, correctif, ce que ça montre.

## Vue d'ensemble

| # | Domaine | Incident | Correctif |
|---|---|---|---|
| 1 | IA, mémoire | Ollama tué (OOMKilled) après 3 heures d'usage | Cache de prompts plafonné à 512 MiB |
| 2 | IA, exploitation | L'évaluation du RAG faisait redémarrer l'API | Évaluation déplacée dans le Job d'indexation |
| 3 | IA, qualité | Recherche à 83 % à cause de titres vagues | Titres explicites, mesure à chaque déploiement |
| 4 | IA, qualité | Régression détectée en ajoutant des documents | Jeu de référence corrigé, mesures conservées dans l'ADR |
| 5 | IA, qualité | Le petit modèle répondait en anglais à des questions en français | Consigne finale écrite dans la langue de la question |
| 6 | IA, capacité | `qwen3:4b` ne tient pas dans une VM de 8 Go | Rester sur `qwen3:1.7b`, Claude via API en option |
| 7 | Sécurité | Le scan Trivy du chart ignorait le Job | Scan du rendu complet `helm template` |
| 8 | Sécurité | Vulnérabilités HIGH dans l'image Python | Plus de `pip` à l'exécution, dépendance épinglée, mises à jour Debian |
| 9 | Sécurité | Appliquer Terraform sans tokens coupait Argo CD du dépôt | Garde-fou `check-tokens` dans le Makefile |
| 10 | Sécurité | Les secrets scellés devenaient illisibles après `make down` | Clé de scellement stable hors du cluster |
| 11 | Chaîne d'approvisionnement | Fichier de checksums de k3d illisible par Ansible | Empreintes SHA-256 épinglées dans le code |
| 12 | Build | Images amd64 émulées sur Mac Apple Silicon | Images multi-architecture amd64 et arm64 |
| 13 | Build | Le modèle embarqué était illisible par l'utilisateur non-root | `chmod a+rX` au build |
| 14 | IaC | Providers Terraform k3d abandonnés depuis 2023 | `terraform_data` et fichier de config k3d |
| 15 | IaC | `kubernetes_manifest` ne planifie pas sans les CRD | Petits charts Helm locaux |
| 16 | IaC | Bloc `moved` refusé entre deux types de ressources | Retrait, vérification que le plan ne recrée pas le cluster |
| 17 | GitOps | Colonne REVISION vide dans `platformctl status` | Lecture de `status.sync.revisions` pour les apps multi-sources |
| 18 | Outillage | `ansible-lint` ne voyait pas les collections | Collections installées dans le projet |
| 19 | Dépendances | Dépôt Helm de Sealed Secrets déplacé | Nouvelle URL officielle |

## IA et RAG

### 1. Ollama tué par manque de mémoire après quelques heures

- **Symptôme** : après environ 3 heures, l'assistant renvoie
  `Ollama request failed: Connection refused`. Le pod Ollama est `OOMKilled`
  à sa limite de 4 GiB, alors que le modèle pèse 1,4 Go.
- **Cause** : le serveur llama.cpp embarqué dans Ollama garde en mémoire un
  cache des prompts déjà traités, plafonné par défaut à 8 GiB, soit plus que la
  limite du conteneur. Chaque question RAG produit un prompt différent (la
  question et ses extraits), donc le cache grossissait d'environ 150 MiB par
  question jusqu'au crash (1,1 GiB après seulement 6 prompts dans les logs). Ollama 0.35 n'expose aucun réglage pour ce cache
  (ticket ollama/ollama#18264).
- **Correctif** : la variable `LLAMA_ARG_CACHE_RAM=512`, lue directement par
  llama.cpp, plafonne le cache à 512 MiB. J'ai aussi fixé explicitement le
  contexte (4096 tokens), une requête et un modèle à la fois. Vérifié après 6
  questions d'affilée : le cache reste à 303 MiB sur 512 MiB, le pod à 2,2 GiB.
- **Ce que ça montre** : une limite mémoire Kubernetes ne suffit pas si le
  logiciel ne connaît pas sa propre consommation. Il faut lire les logs du
  composant (ici `cache state: ... limits: 8192 MiB`) pour trouver le vrai
  consommateur.

### 2. Lancer l'évaluation faisait redémarrer l'API

- **Symptôme** : `kubectl exec ... python -m app.evaluate` se termine en code
  137 et le pod de l'API redémarre.
- **Cause** : l'évaluation charge une deuxième copie du modèle d'embeddings
  dans le même conteneur, déjà limité à 1 GiB. Le noyau tue alors le plus gros
  processus, c'est-à-dire le serveur.
- **Correctif** : l'évaluation tourne maintenant dans le Job d'indexation,
  après chaque déploiement, et publie le résultat dans ses logs.
- **Ce que ça montre** : une commande de diagnostic ne doit jamais pouvoir
  dégrader la production. Les tâches ponctuelles vont dans des Jobs, pas dans
  les conteneurs qui servent du trafic.

### 3. Des titres vagues dégradaient la recherche

- **Symptôme** : à « Comment créer un nouveau service ? », l'assistant
  oubliait la commande `platformctl new-service`.
- **Cause** : la section qui la contenait s'appelait « Scaffold it ». Son
  embedding ne ressemblait pas à la question.
- **Correctif** : un jeu de 12 questions de référence mesure le recall@k. Le
  titre a été renommé en « Create the service with platformctl new-service » :
  le recall@4 passe de 83 % à 100 %.
- **Ce que ça montre** : on améliore un RAG en mesurant, pas à l'intuition. Et
  la qualité de la documentation est la première variable.

### 4. Une régression détectée par l'évaluation

- **Symptôme** : après l'ajout de nouvelles sections aux READMEs, le recall@5
  tombe de 100 % à 92 %.
- **Cause** : la nouvelle section française du README répondait mieux à la
  question française que le guide anglais, mais le jeu de référence
  n'acceptait que le guide.
- **Correctif** : chaque question liste toutes les sections qui y répondent.
  Les quatre mesures, régression comprise, sont gardées dans l'ADR 6.
- **Ce que ça montre** : une évaluation sert aussi à se remettre en question :
  ici le défaut était dans le jeu de test, pas dans la recherche.

### 5. Le modèle répondait en anglais à des questions en français

- **Symptôme** : réponses en anglais, et une fois une commande inventée
  (`platformctl deploy`).
- **Cause** : un modèle de 1,7 milliard de paramètres suit mal une consigne
  générale (« réponds dans la langue de la question ») quand tous les extraits
  sont en anglais.
- **Correctif** : la langue de la question est détectée et le prompt se
  termine par une consigne écrite dans cette langue (« Réponds en français,
  en citant les extraits »). Testé sur 3 questions en français : les 3
  réponses sont en français. Le code accepte aussi Claude via l'API quand la qualité compte.
- **Ce que ça montre** : avec un petit modèle, la position et la langue d'une
  consigne comptent plus que sa formulation.

### 6. Un modèle plus gros ne tient pas dans le budget mémoire

- **Symptôme** : `qwen3:4b` est `OOMKilled` à la première question.
- **Cause** : la VM Colima fait 8 Go et la plateforme en utilise déjà environ
  5.
- **Correctif** : rester sur `qwen3:1.7b` par défaut et documenter les options
  (agrandir la VM ou utiliser une API) avec leurs coûts.
- **Ce que ça montre** : le choix d'un modèle est une décision de capacité,
  pas seulement de qualité.

## Sécurité

### 7. Le scan de sécurité du chart ignorait le Job

- **Symptôme** : Trivy listait tous les templates du chart, sauf `jobs.yaml`.
- **Cause** : Helm classe les Jobs de type *hook* à part du manifeste
  principal, et le scan de chart de Trivy ne regarde que ce manifeste. Un test
  local donnait en plus un faux résultat : Colima ne partage que le dossier
  personnel avec sa VM, donc un dossier dans `/tmp` apparaissait vide au
  conteneur Trivy.
- **Correctif** : la CI scanne le rendu complet de `helm template`, avec toutes
  les capacités activées (`ci/scan-values.yaml`).
- **Ce que ça montre** : un scan vert ne prouve rien tant qu'on n'a pas vérifié
  ce qu'il a réellement analysé.

### 8. Vulnérabilités HIGH dans l'image Python

- **Symptôme** : 5 vulnérabilités HIGH corrigeables dans l'image du RAG.
- **Cause** : `msgpack` et `setuptools` étaient embarqués dans `pip`, inutile à
  l'exécution. `urllib3` arrivait en version vulnérable par une dépendance
  indirecte. `libpcre2` venait de l'image Debian de base.
- **Correctif** : `pip` retiré de l'image finale, `urllib3` épinglé dans la
  version corrigée, mises à jour de sécurité Debian au build. Appliqué aussi au
  template Python : tous les futurs services en profitent. La CI bloque
  désormais toute image avec une vulnérabilité HIGH ou CRITICAL corrigeable.
- **Ce que ça montre** : réduire la surface (ce qui n'est pas dans l'image ne
  peut pas être vulnérable) plutôt qu'empiler les exceptions.

### 9. Appliquer sans tokens coupait Argo CD du dépôt privé

- **Symptôme** : risque identifié avant l'incident : `make platform` lancé
  sans les variables d'environnement remplace le secret du dépôt par un
  secret vide.
- **Cause** : Terraform applique l'état décrit, y compris un token vide.
- **Correctif** : `make cluster` et `make platform` refusent de tourner sans
  les deux tokens, sauf `REQUIRE_TOKENS=0` explicite (utilisé par la CI). Les
  tokens peuvent aussi vivre dans `~/.config/paved-road/env`.
- **Ce que ça montre** : protéger les opérations dangereuses par défaut, et
  rendre le contournement explicite.

### 10. Les secrets scellés devenaient illisibles après une recréation

- **Symptôme** : prévu dès la conception : chaque `make down && make up`
  génère une nouvelle clé Sealed Secrets, donc un secret scellé dans Git ne
  peut plus être déchiffré.
- **Correctif** : la paire de clés est créée une fois dans
  `~/.config/paved-road/sealed-secrets/` et installée par Terraform avant le
  contrôleur. `make fclean` ne la supprime pas, volontairement.
- **Ce que ça montre** : penser au cycle de vie des clés, pas seulement à
  leur usage. En production : Vault et External Secrets (ADR 3).

### 11. Le fichier de checksums de k3d était illisible par Ansible

- **Symptôme** : le module `get_url` ne trouvait pas l'empreinte du binaire
  dans le fichier publié par k3d.
- **Cause** : le fichier contient des chemins (`_dist/k3d-linux-amd64`) au lieu
  de noms de fichiers.
- **Correctif** : les empreintes SHA-256 de chaque binaire et architecture sont
  épinglées dans `ansible/vars/toolchain.yml`.
- **Ce que ça montre** : c'est en fait plus sûr. Un fichier de checksums servi
  par la même source que le binaire ne protège pas d'une source compromise.

## Build et CI

### 12. Images amd64 émulées sur Mac

- **Symptôme** : `hello` tournait sur le cluster arm64 du Mac alors que la CI
  ne construisait que de l'amd64.
- **Cause** : Colima émule l'amd64 de façon transparente. Acceptable pour un
  binaire Go, beaucoup trop lent pour un modèle ONNX en Python.
- **Correctif** : la CI publie des images amd64 et arm64 (buildx, QEMU). Les
  pull requests ne construisent que l'amd64 pour rester rapides.
- **Ce que ça montre** : une équipe mixte (Mac Apple Silicon et serveurs x86)
  a besoin d'images multi-architecture.

### 13. Modèle illisible par l'utilisateur non-root

- **Symptôme** : avertissement `Permission denied` sur les fichiers du modèle
  d'embeddings au démarrage (fastembed ignorait son cache).
- **Cause** : téléchargé au build par root, avec des fichiers en lecture
  réservée au propriétaire. Le conteneur tourne en UID 10001.
- **Correctif** : `chmod -R a+rX` après le téléchargement. Ça rend aussi
  l'image compatible avec OpenShift, qui impose un UID aléatoire (ADR 7).

## Infrastructure as Code

### 14. Providers Terraform k3d abandonnés

- **Cause** : aucune version des providers k3d depuis 2023.
- **Correctif** : le cluster est décrit par un fichier de config k3d généré
  par Terraform et appliqué par une ressource `terraform_data` idempotente
  (ADR 2).
- **Ce que ça montre** : vérifier la maintenance d'une dépendance avant de
  l'adopter, et savoir l'encapsuler proprement quand il n'y a pas d'outil
  maintenu.

### 15. Les ressources personnalisées ne se planifient pas

- **Cause** : `kubernetes_manifest` a besoin que la CRD existe au moment du
  plan, alors que cert-manager et Argo CD l'installent dans le même apply.
- **Correctif** : les ClusterIssuers et l'Application racine sont dans de
  petits charts Helm locaux (`platform-pki`, `gitops`).

### 16. Bloc `moved` refusé

- **Symptôme** : `Unable to Move Resource State` en passant de `local_file` à
  `local_sensitive_file`, nécessaire pour ne pas exposer le token du registre.
- **Correctif** : bloc retiré, et vérification dans le plan que le cluster
  n'était que mis à jour, pas recréé.
- **Ce que ça montre** : toujours lire le plan avant d'appliquer un
  changement qui touche une ressource coûteuse à recréer.

## GitOps et outillage

### 17. Révision vide dans `platformctl status`

- **Cause** : une Application Argo CD à plusieurs sources (le chart partagé et
  le `values.yaml` du service) publie `status.sync.revisions`, un tableau, et
  plus `status.sync.revision`.
- **Correctif** : lecture des deux champs, avec un test unitaire pour chaque
  forme.

### 18. `ansible-lint` ne voyait pas les collections

- **Cause** : `ansible-lint` installé par Homebrew a son propre environnement
  Python et ne voit pas les collections de l'installation d'Ansible.
- **Correctif** : collections installées dans le projet (`ansible/collections`)
  et recherche dans le système désactivée. Le résultat est le même sur toutes
  les machines.

### 19. Dépôt Helm de Sealed Secrets déplacé

- **Symptôme** : `404` sur `bitnami-labs.github.io/sealed-secrets`.
- **Correctif** : nouvelle URL `bitnami.github.io/sealed-secrets`, et le chart
  du projet amont plutôt que celui de Bitnami, dont les images sont devenues
  payantes.

## Questions d'entretien que ce document prépare

- « Racontez un incident en production et comment vous l'avez diagnostiqué. »
  Le 1 (Ollama) : symptôme, logs, cause racine, correctif vérifié sous charge.
- « Comment savez-vous que votre système d'IA fonctionne ? » Les 3 et 4 :
  mesure, amélioration, régression détectée.
- « Comment sécurisez-vous une chaîne CI/CD ? » Les 7, 8 et 11.
- « Une erreur que vous avez faite ? » Le 2 : une commande de diagnostic qui
  a fait redémarrer le service, et pourquoi c'est maintenant impossible.
