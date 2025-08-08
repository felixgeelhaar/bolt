# Terraform Configuration for Monitoring Stack Components
# Deploys Prometheus, Grafana, AlertManager, and Jaeger on EKS

# Kubernetes provider configuration
provider "kubernetes" {
  host                   = aws_eks_cluster.monitoring.endpoint
  cluster_ca_certificate = base64decode(aws_eks_cluster.monitoring.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.monitoring.token
}

provider "helm" {
  kubernetes {
    host                   = aws_eks_cluster.monitoring.endpoint
    cluster_ca_certificate = base64decode(aws_eks_cluster.monitoring.certificate_authority[0].data)
    token                  = data.aws_eks_cluster_auth.monitoring.token
  }
}

data "aws_eks_cluster_auth" "monitoring" {
  name = aws_eks_cluster.monitoring.name
}

# Monitoring namespace
resource "kubernetes_namespace" "monitoring" {
  metadata {
    name = var.namespace
    
    labels = {
      name                                 = var.namespace
      environment                          = var.environment
      "pod-security.kubernetes.io/enforce" = "restricted"
      "pod-security.kubernetes.io/audit"   = "restricted"
      "pod-security.kubernetes.io/warn"    = "restricted"
    }
  }

  depends_on = [aws_eks_node_group.monitoring]
}

# Grafana admin password secret
resource "kubernetes_secret" "grafana_admin" {
  metadata {
    name      = "grafana-admin-secret"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
  }

  data = {
    admin-password = "bolt-monitoring-2024-secure"
  }

  type = "Opaque"
}

# StorageClass for monitoring components
resource "kubernetes_storage_class" "gp3" {
  metadata {
    name = "gp3-monitoring"
    
    annotations = {
      "storageclass.kubernetes.io/is-default-class" = "false"
    }
  }
  
  storage_provisioner    = "ebs.csi.aws.com"
  reclaim_policy        = "Delete"
  volume_binding_mode   = "WaitForFirstConsumer"
  allow_volume_expansion = true
  
  parameters = {
    type      = "gp3"
    fsType    = "ext4"
    encrypted = "true"
  }
}

# Persistent Volumes for monitoring data
resource "kubernetes_persistent_volume_claim" "prometheus_data" {
  metadata {
    name      = "prometheus-data"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
  }
  
  spec {
    access_modes = ["ReadWriteOnce"]
    storage_class_name = kubernetes_storage_class.gp3.metadata[0].name
    
    resources {
      requests = {
        storage = "100Gi"
      }
    }
  }
}

resource "kubernetes_persistent_volume_claim" "grafana_data" {
  metadata {
    name      = "grafana-data"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
  }
  
  spec {
    access_modes = ["ReadWriteOnce"]
    storage_class_name = kubernetes_storage_class.gp3.metadata[0].name
    
    resources {
      requests = {
        storage = "10Gi"
      }
    }
  }
}

# ConfigMaps for monitoring configurations
resource "kubernetes_config_map" "prometheus_config" {
  metadata {
    name      = "prometheus-config"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
  }

  data = {
    "prometheus.yml" = file("${path.module}/../monitoring/prometheus/prometheus.yml")
  }
}

resource "kubernetes_config_map" "prometheus_rules" {
  metadata {
    name      = "prometheus-rules"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
  }

  data = {
    "bolt-recording.yml"   = file("${path.module}/../monitoring/prometheus/rules/bolt-recording.yml")
    "bolt-performance.yml" = file("${path.module}/../monitoring/prometheus/alerts/bolt-performance.yml")
  }
}

resource "kubernetes_config_map" "alertmanager_config" {
  metadata {
    name      = "alertmanager-config"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
  }

  data = {
    "alertmanager.yml" = templatefile("${path.module}/../monitoring/alertmanager/alertmanager.yml", {
      slack_webhook_url = var.alertmanager_slack_webhook
    })
  }
}

resource "kubernetes_config_map" "alertmanager_templates" {
  metadata {
    name      = "alertmanager-templates"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
  }

  data = {
    "bolt-alerts.tmpl" = file("${path.module}/../monitoring/alertmanager/templates/bolt-alerts.tmpl")
  }
}

resource "kubernetes_config_map" "grafana_dashboards" {
  metadata {
    name      = "grafana-dashboards"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    
    labels = {
      grafana_dashboard = "1"
    }
  }

  data = {
    "bolt-performance-overview.json" = file("${path.module}/../monitoring/grafana/dashboards/bolt-performance-overview.json")
    "bolt-operational-health.json"   = file("${path.module}/../monitoring/grafana/dashboards/bolt-operational-health.json")
  }
}

# Service Accounts with proper RBAC
resource "kubernetes_service_account" "prometheus" {
  metadata {
    name      = "prometheus"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
  }
  
  automount_service_account_token = false
}

resource "kubernetes_cluster_role" "prometheus" {
  metadata {
    name = "prometheus"
  }
  
  rule {
    api_groups = [""]
    resources  = ["nodes", "nodes/proxy", "services", "endpoints", "pods"]
    verbs      = ["get", "list", "watch"]
  }
  
  rule {
    api_groups = ["extensions"]
    resources  = ["ingresses"]
    verbs      = ["get", "list", "watch"]
  }
  
  rule {
    non_resource_urls = ["/metrics"]
    verbs             = ["get"]
  }
}

resource "kubernetes_cluster_role_binding" "prometheus" {
  metadata {
    name = "prometheus"
  }
  
  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = kubernetes_cluster_role.prometheus.metadata[0].name
  }
  
  subject {
    kind      = "ServiceAccount"
    name      = kubernetes_service_account.prometheus.metadata[0].name
    namespace = kubernetes_namespace.monitoring.metadata[0].name
  }
}

# Prometheus Deployment
resource "kubernetes_deployment" "prometheus" {
  metadata {
    name      = "prometheus"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    
    labels = {
      app = "prometheus"
    }
  }
  
  spec {
    replicas = 2
    
    selector {
      match_labels = {
        app = "prometheus"
      }
    }
    
    template {
      metadata {
        labels = {
          app = "prometheus"
        }
        
        annotations = {
          "prometheus.io/scrape" = "true"
          "prometheus.io/port"   = "9090"
        }
      }
      
      spec {
        service_account_name            = kubernetes_service_account.prometheus.metadata[0].name
        automount_service_account_token = false
        
        security_context {
          run_as_non_root = true
          run_as_user     = 65534
          run_as_group    = 65534
          fs_group        = 65534
        }
        
        # Toleration for monitoring nodes
        toleration {
          key    = "monitoring"
          value  = "true"
          effect = "NoSchedule"
        }
        
        # Node affinity for monitoring nodes
        affinity {
          node_affinity {
            preferred_during_scheduling_ignored_during_execution {
              weight = 100
              
              preference {
                match_expressions {
                  key      = "node-type"
                  operator = "In"
                  values   = ["monitoring"]
                }
              }
            }
          }
          
          # Anti-affinity to spread replicas
          pod_anti_affinity {
            preferred_during_scheduling_ignored_during_execution {
              weight = 100
              
              pod_affinity_term {
                label_selector {
                  match_expressions {
                    key      = "app"
                    operator = "In"
                    values   = ["prometheus"]
                  }
                }
                topology_key = "kubernetes.io/hostname"
              }
            }
          }
        }
        
        container {
          name  = "prometheus"
          image = "prom/prometheus:v2.47.0"
          
          security_context {
            allow_privilege_escalation = false
            capabilities {
              drop = ["ALL"]
            }
            read_only_root_filesystem = true
            run_as_non_root          = true
          }
          
          port {
            container_port = 9090
            name          = "http"
          }
          
          args = [
            "--config.file=/etc/prometheus/prometheus.yml",
            "--storage.tsdb.path=/prometheus",
            "--storage.tsdb.retention.time=${var.prometheus_retention}",
            "--storage.tsdb.retention.size=50GB",
            "--web.console.libraries=/etc/prometheus/console_libraries",
            "--web.console.templates=/etc/prometheus/consoles",
            "--web.enable-lifecycle",
            "--web.enable-admin-api",
            "--log.level=info"
          ]
          
          volume_mount {
            name       = "prometheus-config"
            mount_path = "/etc/prometheus"
          }
          
          volume_mount {
            name       = "prometheus-rules"
            mount_path = "/etc/prometheus/rules"
          }
          
          volume_mount {
            name       = "prometheus-data"
            mount_path = "/prometheus"
          }
          
          liveness_probe {
            http_get {
              path = "/-/healthy"
              port = 9090
            }
            initial_delay_seconds = 30
            period_seconds        = 15
          }
          
          readiness_probe {
            http_get {
              path = "/-/ready"
              port = 9090
            }
            initial_delay_seconds = 5
            period_seconds        = 5
          }
          
          resources {
            requests = {
              cpu    = "500m"
              memory = "1Gi"
            }
            limits = {
              cpu    = "2000m"
              memory = "4Gi"
            }
          }
        }
        
        volume {
          name = "prometheus-config"
          config_map {
            name = kubernetes_config_map.prometheus_config.metadata[0].name
          }
        }
        
        volume {
          name = "prometheus-rules"
          config_map {
            name = kubernetes_config_map.prometheus_rules.metadata[0].name
          }
        }
        
        volume {
          name = "prometheus-data"
          persistent_volume_claim {
            claim_name = kubernetes_persistent_volume_claim.prometheus_data.metadata[0].name
          }
        }
      }
    }
  }
  
  depends_on = [
    kubernetes_config_map.prometheus_config,
    kubernetes_config_map.prometheus_rules,
    kubernetes_persistent_volume_claim.prometheus_data
  ]
}

# Prometheus Service
resource "kubernetes_service" "prometheus" {
  metadata {
    name      = "prometheus"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    
    labels = {
      app = "prometheus"
    }
    
    annotations = {
      "prometheus.io/scrape" = "true"
      "prometheus.io/port"   = "9090"
    }
  }
  
  spec {
    selector = {
      app = "prometheus"
    }
    
    port {
      name        = "http"
      port        = 9090
      target_port = 9090
      protocol    = "TCP"
    }
    
    type = "ClusterIP"
  }
}

# AlertManager Deployment
resource "kubernetes_deployment" "alertmanager" {
  metadata {
    name      = "alertmanager"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    
    labels = {
      app = "alertmanager"
    }
  }
  
  spec {
    replicas = 3
    
    selector {
      match_labels = {
        app = "alertmanager"
      }
    }
    
    template {
      metadata {
        labels = {
          app = "alertmanager"
        }
      }
      
      spec {
        security_context {
          run_as_non_root = true
          run_as_user     = 65534
          run_as_group    = 65534
          fs_group        = 65534
        }
        
        # Toleration for monitoring nodes
        toleration {
          key    = "monitoring"
          value  = "true"
          effect = "NoSchedule"
        }
        
        # Anti-affinity to spread replicas
        affinity {
          pod_anti_affinity {
            required_during_scheduling_ignored_during_execution {
              label_selector {
                match_expressions {
                  key      = "app"
                  operator = "In"
                  values   = ["alertmanager"]
                }
              }
              topology_key = "kubernetes.io/hostname"
            }
          }
        }
        
        container {
          name  = "alertmanager"
          image = "prom/alertmanager:v0.26.0"
          
          security_context {
            allow_privilege_escalation = false
            capabilities {
              drop = ["ALL"]
            }
            read_only_root_filesystem = true
            run_as_non_root          = true
          }
          
          port {
            container_port = 9093
            name          = "http"
          }
          
          args = [
            "--config.file=/etc/alertmanager/alertmanager.yml",
            "--storage.path=/alertmanager",
            "--web.external-url=http://alertmanager:9093",
            "--cluster.listen-address=0.0.0.0:9094",
            "--cluster.advertise-address=$(POD_IP):9094",
            "--log.level=info"
          ]
          
          env {
            name = "POD_IP"
            value_from {
              field_ref {
                field_path = "status.podIP"
              }
            }
          }
          
          volume_mount {
            name       = "alertmanager-config"
            mount_path = "/etc/alertmanager"
          }
          
          volume_mount {
            name       = "alertmanager-templates"
            mount_path = "/etc/alertmanager/templates"
          }
          
          volume_mount {
            name       = "alertmanager-data"
            mount_path = "/alertmanager"
          }
          
          liveness_probe {
            http_get {
              path = "/-/healthy"
              port = 9093
            }
            initial_delay_seconds = 30
            period_seconds        = 15
          }
          
          readiness_probe {
            http_get {
              path = "/-/ready"
              port = 9093
            }
            initial_delay_seconds = 5
            period_seconds        = 5
          }
          
          resources {
            requests = {
              cpu    = "100m"
              memory = "256Mi"
            }
            limits = {
              cpu    = "500m"
              memory = "512Mi"
            }
          }
        }
        
        volume {
          name = "alertmanager-config"
          config_map {
            name = kubernetes_config_map.alertmanager_config.metadata[0].name
          }
        }
        
        volume {
          name = "alertmanager-templates"
          config_map {
            name = kubernetes_config_map.alertmanager_templates.metadata[0].name
          }
        }
        
        volume {
          name = "alertmanager-data"
          empty_dir {}
        }
      }
    }
  }
  
  depends_on = [
    kubernetes_config_map.alertmanager_config,
    kubernetes_config_map.alertmanager_templates
  ]
}

# AlertManager Service
resource "kubernetes_service" "alertmanager" {
  metadata {
    name      = "alertmanager"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    
    labels = {
      app = "alertmanager"
    }
  }
  
  spec {
    selector = {
      app = "alertmanager"
    }
    
    port {
      name        = "http"
      port        = 9093
      target_port = 9093
      protocol    = "TCP"
    }
    
    port {
      name        = "cluster"
      port        = 9094
      target_port = 9094
      protocol    = "TCP"
    }
    
    type = "ClusterIP"
  }
}

# Grafana Deployment
resource "kubernetes_deployment" "grafana" {
  metadata {
    name      = "grafana"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    
    labels = {
      app = "grafana"
    }
  }
  
  spec {
    replicas = 1
    
    selector {
      match_labels = {
        app = "grafana"
      }
    }
    
    template {
      metadata {
        labels = {
          app = "grafana"
        }
      }
      
      spec {
        security_context {
          run_as_non_root = true
          run_as_user     = 472
          run_as_group    = 0
          fs_group        = 472
        }
        
        # Toleration for monitoring nodes
        toleration {
          key    = "monitoring"
          value  = "true"
          effect = "NoSchedule"
        }
        
        container {
          name  = "grafana"
          image = "grafana/grafana:10.1.0"
          
          security_context {
            allow_privilege_escalation = false
            capabilities {
              drop = ["ALL"]
            }
            read_only_root_filesystem = false
            run_as_non_root          = true
          }
          
          port {
            container_port = 3000
            name          = "http"
          }
          
          env {
            name = "GF_SECURITY_ADMIN_PASSWORD"
            value_from {
              secret_key_ref {
                name = kubernetes_secret.grafana_admin.metadata[0].name
                key  = "admin-password"
              }
            }
          }
          
          env {
            name  = "GF_INSTALL_PLUGINS"
            value = "grafana-piechart-panel,grafana-worldmap-panel"
          }
          
          env {
            name  = "GF_DASHBOARDS_DEFAULT_HOME_DASHBOARD_PATH"
            value = "/var/lib/grafana/dashboards/bolt-performance-overview.json"
          }
          
          volume_mount {
            name       = "grafana-data"
            mount_path = "/var/lib/grafana"
          }
          
          volume_mount {
            name       = "grafana-dashboards"
            mount_path = "/var/lib/grafana/dashboards"
          }
          
          liveness_probe {
            http_get {
              path = "/api/health"
              port = 3000
            }
            initial_delay_seconds = 60
            period_seconds        = 30
          }
          
          readiness_probe {
            http_get {
              path = "/api/health"
              port = 3000
            }
            initial_delay_seconds = 10
            period_seconds        = 5
          }
          
          resources {
            requests = {
              cpu    = "100m"
              memory = "256Mi"
            }
            limits = {
              cpu    = "500m"
              memory = "512Mi"
            }
          }
        }
        
        volume {
          name = "grafana-data"
          persistent_volume_claim {
            claim_name = kubernetes_persistent_volume_claim.grafana_data.metadata[0].name
          }
        }
        
        volume {
          name = "grafana-dashboards"
          config_map {
            name = kubernetes_config_map.grafana_dashboards.metadata[0].name
          }
        }
      }
    }
  }
  
  depends_on = [
    kubernetes_config_map.grafana_dashboards,
    kubernetes_persistent_volume_claim.grafana_data
  ]
}

# Grafana Service
resource "kubernetes_service" "grafana" {
  metadata {
    name      = "grafana"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    
    labels = {
      app = "grafana"
    }
  }
  
  spec {
    selector = {
      app = "grafana"
    }
    
    port {
      name        = "http"
      port        = 3000
      target_port = 3000
      protocol    = "TCP"
    }
    
    type = "ClusterIP"
  }
}

# Network Policies for secure communication
resource "kubernetes_network_policy" "monitoring_default_deny" {
  metadata {
    name      = "default-deny-all"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
  }

  spec {
    pod_selector {}
    policy_types = ["Ingress", "Egress"]
  }
}

resource "kubernetes_network_policy" "prometheus_ingress" {
  metadata {
    name      = "prometheus-ingress"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
  }

  spec {
    pod_selector {
      match_labels = {
        app = "prometheus"
      }
    }

    policy_types = ["Ingress"]

    ingress {
      from {
        pod_selector {
          match_labels = {
            app = "grafana"
          }
        }
      }
      ports {
        port     = "9090"
        protocol = "TCP"
      }
    }
  }
}

resource "kubernetes_network_policy" "grafana_ingress" {
  metadata {
    name      = "grafana-ingress"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
  }

  spec {
    pod_selector {
      match_labels = {
        app = "grafana"
      }
    }

    policy_types = ["Ingress", "Egress"]

    ingress {
      ports {
        port     = "3000"
        protocol = "TCP"
      }
    }
    
    egress {
      to {
        pod_selector {
          match_labels = {
            app = "prometheus"
          }
        }
      }
      ports {
        port     = "9090"
        protocol = "TCP"
      }
    }
  }
}

# Outputs for monitoring stack (updated for ClusterIP services)
output "prometheus_endpoint" {
  description = "Prometheus service endpoint (ClusterIP)"
  value       = "http://${kubernetes_service.prometheus.metadata[0].name}.${kubernetes_namespace.monitoring.metadata[0].name}.svc.cluster.local:9090"
}

output "grafana_endpoint" {
  description = "Grafana service endpoint (ClusterIP)"  
  value       = "http://${kubernetes_service.grafana.metadata[0].name}.${kubernetes_namespace.monitoring.metadata[0].name}.svc.cluster.local:3000"
}

output "alertmanager_endpoint" {
  description = "AlertManager service endpoint (ClusterIP)"
  value       = "http://${kubernetes_service.alertmanager.metadata[0].name}.${kubernetes_namespace.monitoring.metadata[0].name}.svc.cluster.local:9093"
}