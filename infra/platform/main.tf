locals {
  # Settings shared with platformctl and the CI.
  config = yamldecode(file("${path.module}/../../platform/config.yaml"))

  # Every platform UI gets a certificate from the internal CA through this annotation.
  cluster_issuer = "paved-road-ca"

  full = var.profile == "full"

  # Profile-specific values are layered on top of the base values file.
  kube_prometheus_stack_values = local.full ? ["kube-prometheus-stack-logs.yaml"] : ["kube-prometheus-stack-lite.yaml"]
  argo_cd_values               = local.full ? [] : ["argo-cd-lite.yaml"]
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
  values = [
    for f in concat(["kube-prometheus-stack.yaml"], local.kube_prometheus_stack_values) :
    file("${path.module}/values/${f}")
  ]
  timeout = 600

  depends_on = [helm_release.platform_pki, kubernetes_secret_v1.grafana_admin]
}

# The log pipeline is only part of the full profile.
resource "helm_release" "loki" {
  count = local.full ? 1 : 0

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
  count = local.full ? 1 : 0

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
  values = [
    for f in concat(["argo-cd.yaml"], local.argo_cd_values) :
    file("${path.module}/values/${f}")
  ]
  timeout = 600

  # ServiceMonitor CRDs come from kube-prometheus-stack.
  depends_on = [helm_release.platform_pki, helm_release.kube_prometheus_stack]
}

# Connects Argo CD to the repository and hands platform/argocd over to it.
resource "helm_release" "gitops" {
  name      = "gitops"
  namespace = "argocd"
  chart     = "${path.module}/charts/gitops"

  set = [
    { name = "repoURL", value = local.config.repoURL },
    { name = "revision", value = local.config.revision },
    { name = "username", value = "paved-road" },
  ]
  set_sensitive = [{ name = "token", value = var.git_token }]

  depends_on = [helm_release.argo_cd]
}

moved {
  from = helm_release.loki
  to   = helm_release.loki[0]
}

moved {
  from = helm_release.alloy
  to   = helm_release.alloy[0]
}
