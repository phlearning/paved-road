SHELL := /bin/bash
.DEFAULT_GOAL := help

CLUSTER_NAME ?= paved-road
KUBECONFIG_PATH := $(CURDIR)/.kube/config
TF_CLUSTER := infra/cluster
TF_PLATFORM := infra/platform

export KUBECONFIG := $(KUBECONFIG_PATH)

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: bootstrap
bootstrap: ## Install the toolchain and start the container runtime (Ansible)
	@command -v ansible-playbook >/dev/null || brew install ansible
	cd ansible && ansible-galaxy collection install -r requirements.yml -p ./collections
	cd ansible && ansible-playbook bootstrap.yml

.PHONY: up
up: cluster platform ## Create the cluster and install the platform
	@$(MAKE) --no-print-directory urls

.PHONY: cluster
cluster: ## Create the k3d cluster (Terraform)
	terraform -chdir=$(TF_CLUSTER) init -input=false
	terraform -chdir=$(TF_CLUSTER) apply -input=false -auto-approve \
		-var cluster_name=$(CLUSTER_NAME) -var kubeconfig_path=$(KUBECONFIG_PATH)

.PHONY: platform
platform: ## Install the platform components (Terraform + Helm)
	terraform -chdir=$(TF_PLATFORM) init -input=false
	terraform -chdir=$(TF_PLATFORM) apply -input=false -auto-approve \
		-var kubeconfig_path=$(KUBECONFIG_PATH)

.PHONY: down
down: ## Destroy the cluster
	-terraform -chdir=$(TF_CLUSTER) destroy -input=false -auto-approve \
		-var cluster_name=$(CLUSTER_NAME) -var kubeconfig_path=$(KUBECONFIG_PATH)
	rm -f $(TF_PLATFORM)/terraform.tfstate $(TF_PLATFORM)/terraform.tfstate.backup

.PHONY: ca
ca: ## Export the internal root CA to .kube/paved-road-ca.crt
	@kubectl -n cert-manager get secret paved-road-root-ca -o jsonpath='{.data.ca\.crt}' \
		| base64 -d > .kube/paved-road-ca.crt
	@echo "CA written to .kube/paved-road-ca.crt"

.PHONY: trust
trust: ca ## Trust the internal root CA in the macOS keychain (asks for sudo)
	sudo security add-trusted-cert -d -r trustRoot \
		-k /Library/Keychains/System.keychain .kube/paved-road-ca.crt

.PHONY: untrust
untrust: ## Remove the internal root CA from the macOS keychain (asks for sudo)
	sudo security delete-certificate -c "paved-road root CA" /Library/Keychains/System.keychain

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

.PHONY: lint
lint: ## Lint Terraform and Ansible code
	terraform fmt -check -recursive infra
	terraform -chdir=$(TF_CLUSTER) validate
	terraform -chdir=$(TF_PLATFORM) validate
	cd ansible && ansible-lint bootstrap.yml
