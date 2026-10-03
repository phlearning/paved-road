locals {
  # Every platform UI gets a certificate from the internal CA through this annotation.
  cluster_issuer = "paved-road-ca"
}

# --- Secrets -----------------------------------------------------------------

resource "helm_release" "sealed_secrets" {
  name       = "sealed-secrets"
  namespace  = "kube-system"
  repository = "https://bitnami.github.io/sealed-secrets"
  chart      = "sealed-secrets"
  version    = var.chart_versions.sealed_secrets
  values     = [file("${path.module}/values/sealed-secrets.yaml")]
}

# --- PKI ---------------------------------------------------------------------

resource "helm_release" "cert_manager" {
  name             = "cert-manager"
  namespace        = "cert-manager"
  create_namespace = true
  repository       = "https://charts.jetstack.io"
  chart            = "cert-manager"
  version          = var.chart_versions.cert_manager
  values           = [file("${path.module}/values/cert-manager.yaml")]
}

# Issuers live in a local chart because their CRDs only exist once
# cert-manager is installed, which kubernetes_manifest cannot plan around.
resource "helm_release" "platform_pki" {
  name      = "platform-pki"
  namespace = "cert-manager"
  chart     = "${path.module}/charts/platform-pki"

  set = [{
    name  = "clusterIssuer"
    value = local.cluster_issuer
  }]

  depends_on = [helm_release.cert_manager]
}

# --- Observability -----------------------------------------------------------

resource "kubernetes_namespace_v1" "monitoring" {
  metadata {
    name = "monitoring"
  }
}

resource "random_password" "grafana_admin" {
  length  = 24
  special = false
}

resource "kubernetes_secret_v1" "grafana_admin" {
  metadata {
    name      = "grafana-admin"
    namespace = kubernetes_namespace_v1.monitoring.metadata[0].name
  }

  data = {
    admin-user     = "admin"
    admin-password = random_password.grafana_admin.result
  }
}

resource "helm_release" "kube_prometheus_stack" {
  name       = "kube-prometheus-stack"
  namespace  = kubernetes_namespace_v1.monitoring.metadata[0].name
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "kube-prometheus-stack"
  version    = var.chart_versions.kube_prometheus_stack
  values     = [file("${path.module}/values/kube-prometheus-stack.yaml")]
  timeout    = 600

  depends_on = [helm_release.platform_pki, kubernetes_secret_v1.grafana_admin]
}

resource "helm_release" "loki" {
  name       = "loki"
  namespace  = kubernetes_namespace_v1.monitoring.metadata[0].name
  repository = "https://grafana.github.io/helm-charts"
  chart      = "loki"
  version    = var.chart_versions.loki
  values     = [file("${path.module}/values/loki.yaml")]
  timeout    = 600

  depends_on = [helm_release.kube_prometheus_stack]
}

resource "helm_release" "alloy" {
  name       = "alloy"
  namespace  = kubernetes_namespace_v1.monitoring.metadata[0].name
  repository = "https://grafana.github.io/helm-charts"
  chart      = "alloy"
  version    = var.chart_versions.alloy
  values     = [file("${path.module}/values/alloy.yaml")]

  depends_on = [helm_release.loki]
}

# --- Delivery ----------------------------------------------------------------

resource "helm_release" "argo_cd" {
  name             = "argo-cd"
  namespace        = "argocd"
  create_namespace = true
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = var.chart_versions.argo_cd
  values           = [file("${path.module}/values/argo-cd.yaml")]
  timeout          = 600

  # ServiceMonitor CRDs come from kube-prometheus-stack.
  depends_on = [helm_release.platform_pki, helm_release.kube_prometheus_stack]
}
