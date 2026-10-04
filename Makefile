SHELL := /bin/bash
.DEFAULT_GOAL := help

CLUSTER_NAME ?= paved-road
KUBECONFIG_PATH := $(CURDIR)/.kube/config
TF_CLUSTER := infra/cluster
TF_PLATFORM := infra/platform

# Resource profile chosen by `make bootstrap` from the host memory (full or lite).
-include .paved-road.env
PROFILE ?= full

# Ask for the sudo password on Linux unless sudo is passwordless (CI, cloud VMs).
BECOME_FLAGS ?= $(shell [ "$$(uname -s)" = Linux ] && ! sudo -n true 2>/dev/null && echo --ask-become-pass)

# Stable Sealed Secrets key pair, kept outside the cluster (see docs/adr/0003).
SEALING_KEY_DIR ?= $(HOME)/.config/paved-road/sealed-secrets

TF_VARS := -var kubeconfig_path=$(KUBECONFIG_PATH) -var profile=$(PROFILE)

# Read-only credentials for the private repository, taken from the environment
# so they never land in Git (see docs/adr/0005-gitops-delivery.md):
#   PAVED_ROAD_GIT_TOKEN       fine-grained token, Contents: read on the repository
#   PAVED_ROAD_REGISTRY_TOKEN  classic token with read:packages, to pull images
# They can also live in ~/.config/paved-road/env (KEY=value lines, chmod 600).
-include $(HOME)/.config/paved-road/env

# Applying without the tokens would remove Argo CD's access to the private
# repository, so it is refused unless explicitly allowed (the CI does).
REQUIRE_TOKENS ?= 1

export TF_VAR_git_token := $(PAVED_ROAD_GIT_TOKEN)
export TF_VAR_registry_token := $(PAVED_ROAD_REGISTRY_TOKEN)

export KUBECONFIG := $(KUBECONFIG_PATH)
# pipx installs Ansible there on Linux.
export PATH := $(HOME)/.local/bin:$(PATH)

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-z0-9-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: bootstrap
bootstrap: ## Install the toolchain and start the container runtime (Ansible)
	@scripts/ensure-ansible.sh
	cd ansible && ansible-galaxy collection install -r requirements.yml -p ./collections
	cd ansible && ansible-playbook bootstrap.yml $(BECOME_FLAGS) \
		$(if $(filter command line environment,$(origin PROFILE)),-e profile=$(PROFILE))

.PHONY: up
up: cluster platform ## Create the cluster and install the platform (PROFILE=full|lite)
	@$(MAKE) --no-print-directory urls

.PHONY: check-tokens
check-tokens:
	@if [ "$(REQUIRE_TOKENS)" = 1 ] && { [ -z "$(PAVED_ROAD_GIT_TOKEN)" ] || [ -z "$(PAVED_ROAD_REGISTRY_TOKEN)" ]; }; then \
		echo "PAVED_ROAD_GIT_TOKEN and PAVED_ROAD_REGISTRY_TOKEN must be set (environment or ~/.config/paved-road/env)."; \
		echo "For a public repository or a throwaway cluster, run with REQUIRE_TOKENS=0."; \
		exit 1; \
	fi

.PHONY: cluster
cluster: check-tokens ## Create the k3d cluster (Terraform)
	terraform -chdir=$(TF_CLUSTER) init -input=false
	terraform -chdir=$(TF_CLUSTER) apply -input=false -auto-approve \
		-var cluster_name=$(CLUSTER_NAME) $(TF_VARS)

.PHONY: platform
platform: check-tokens sealing-key ## Install the platform components (Terraform + Helm)
	terraform -chdir=$(TF_PLATFORM) init -input=false
	terraform -chdir=$(TF_PLATFORM) apply -input=false -auto-approve $(TF_VARS) \
		-var sealing_key_dir=$(SEALING_KEY_DIR)

.PHONY: sealing-key
sealing-key: ## Create the stable Sealed Secrets key pair once
	@if [ ! -f "$(SEALING_KEY_DIR)/tls.key" ]; then \
		mkdir -p "$(SEALING_KEY_DIR)" && chmod 700 "$(SEALING_KEY_DIR)"; \
		openssl req -x509 -nodes -newkey rsa:4096 -days 3650 \
			-subj "/CN=sealed-secret/O=paved-road" \
			-keyout "$(SEALING_KEY_DIR)/tls.key" -out "$(SEALING_KEY_DIR)/tls.crt" 2>/dev/null; \
		chmod 600 "$(SEALING_KEY_DIR)/tls.key"; \
		echo "Sealed Secrets key pair created in $(SEALING_KEY_DIR)"; \
	fi

.PHONY: down
down: ## Destroy the cluster
	-terraform -chdir=$(TF_CLUSTER) destroy -input=false -auto-approve \
		-var cluster_name=$(CLUSTER_NAME) $(TF_VARS)
	rm -f $(TF_PLATFORM)/terraform.tfstate $(TF_PLATFORM)/terraform.tfstate.backup

.PHONY: ca
ca: ## Export the internal root CA to .kube/paved-road-ca.crt
	@kubectl -n cert-manager get secret paved-road-root-ca -o jsonpath='{.data.ca\.crt}' \
		| base64 -d > .kube/paved-road-ca.crt
	@echo "CA written to .kube/paved-road-ca.crt"

.PHONY: trust
trust: ca ## Trust the internal root CA on this machine (asks for sudo)
	scripts/trust-ca.sh .kube/paved-road-ca.crt

.PHONY: untrust
untrust: ## Remove the internal root CA from this machine (asks for sudo)
	scripts/trust-ca.sh .kube/paved-road-ca.crt --remove

.PHONY: creds
creds: ## Print the admin credentials of the platform UIs
	@echo "Argo CD  admin / $$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d)"
	@echo "Grafana  admin / $$(kubectl -n monitoring get secret grafana-admin -o jsonpath='{.data.admin-password}' | base64 -d)"

.PHONY: urls
urls: ## Print the platform URLs
	@echo "Argo CD  https://argocd.localhost"
	@echo "Grafana  https://grafana.localhost"
	@echo "Run 'make trust' once so your browser accepts the internal CA, and 'make creds' for passwords."

.PHONY: status
status: ## Show the state of the platform workloads
	kubectl get pods -A -o wide | grep -v Completed

.PHONY: cli
cli: ## Build platformctl into bin/
	go build -o bin/platformctl ./cli/cmd/platformctl

.PHONY: test
test: ## Run the Go tests and the chart lint
	go vet ./...
	go test ./...
	helm lint platform/charts/service --set name=lint --set-string image.repository=lint,image.tag=lint

.PHONY: e2e
e2e: ## Scaffold, build and deploy a service per language on the running cluster
	scripts/e2e.sh

.PHONY: lint
lint: ## Lint Terraform and Ansible code
	terraform fmt -check -recursive infra
	terraform -chdir=$(TF_CLUSTER) init -backend=false -input=false >/dev/null
	terraform -chdir=$(TF_CLUSTER) validate
	terraform -chdir=$(TF_PLATFORM) init -backend=false -input=false >/dev/null
	terraform -chdir=$(TF_PLATFORM) validate
	cd ansible && ansible-lint bootstrap.yml tasks/ vars/ handlers/
	shellcheck scripts/*.sh
