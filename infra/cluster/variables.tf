variable "cluster_name" {
  description = "Name of the k3d cluster."
  type        = string
  default     = "paved-road"
}

variable "k3s_image" {
  description = "k3s node image, pinned for reproducibility."
  type        = string
  default     = "rancher/k3s:v1.36.4-k3s1"
}

variable "agents" {
  description = "Number of k3d agent nodes."
  type        = number
  default     = 1
}

variable "kubeconfig_path" {
  description = "Where to write the cluster kubeconfig."
  type        = string
}
