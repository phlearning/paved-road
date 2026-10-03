output "kubeconfig_path" {
  description = "Path of the generated kubeconfig."
  value       = terraform_data.cluster.output.kubeconfig_path
}
