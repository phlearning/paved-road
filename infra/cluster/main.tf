terraform {
  required_version = ">= 1.9"

  required_providers {
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

# The k3d Terraform providers are unmaintained, so the cluster is described
# by a declarative k3d config file and reconciled with the k3d CLI.
# See docs/adr/0002-k3d-via-terraform-data.md.
resource "local_file" "k3d_config" {
  filename = "${path.module}/.generated/k3d.yaml"
  content = templatefile("${path.module}/k3d.yaml.tftpl", {
    cluster_name = var.cluster_name
    k3s_image    = var.k3s_image
    agents       = var.profile == "full" ? 1 : 0
  })
}

resource "terraform_data" "cluster" {
  triggers_replace = [local_file.k3d_config.content]

  input = {
    cluster_name    = var.cluster_name
    config_path     = local_file.k3d_config.filename
    kubeconfig_path = var.kubeconfig_path
  }

  provisioner "local-exec" {
    command     = <<-EOT
      set -euo pipefail
      if ! k3d cluster get ${self.input.cluster_name} >/dev/null 2>&1; then
        k3d cluster create --config ${self.input.config_path}
      fi
      mkdir -p "$(dirname ${self.input.kubeconfig_path})"
      k3d kubeconfig get ${self.input.cluster_name} > ${self.input.kubeconfig_path}
      chmod 600 ${self.input.kubeconfig_path}
    EOT
    interpreter = ["/bin/bash", "-c"]
  }

  provisioner "local-exec" {
    when        = destroy
    command     = "k3d cluster delete ${self.input.cluster_name} && rm -f ${self.input.kubeconfig_path}"
    interpreter = ["/bin/bash", "-c"]
  }
}
