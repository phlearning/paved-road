variable "kubeconfig_path" {
  description = "Kubeconfig of the target cluster (written by infra/cluster)."
  type        = string
}

variable "profile" {
  description = "Resource profile: 'full' or 'lite' (no log pipeline, no Alertmanager, smaller requests)."
  type        = string
  default     = "full"

  validation {
    condition     = contains(["full", "lite"], var.profile)
    error_message = "profile must be 'full' or 'lite'."
  }
}

variable "chart_versions" {
  description = "Pinned Helm chart versions of the platform components."
  type        = map(string)
  default = {
    sealed_secrets        = "2.20.0"
    cert_manager          = "v1.21.2"
    kube_prometheus_stack = "91.9.0"
    loki                  = "7.3.0"
    alloy                 = "1.13.0"
    argo_cd               = "10.9.6"
  }
}
