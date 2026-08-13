terraform {
    backend "gcs" {
        bucket = "h4i-applications-terraform-state"
        prefix = "professor/prod"
    }
}

locals {
  professor_alert_notification_channels = [
    "projects/h4i-applications/notificationChannels/8006514687676133218",
    "projects/h4i-applications/notificationChannels/13261586194715419281",
    "projects/h4i-applications/notificationChannels/12677533325804840933",
  ]
}

resource "google_secret_manager_secret" "professor_github_pat" {
  project             = "h4i-applications"
  secret_id           = "PROFESSOR_GITHUB_PAT"
  replication {
    auto {
    }
  }
}

resource "google_service_account" "professor_service" {
  account_id      = "professor-service"
  deletion_policy = "ABANDON"
  description     = "Role for Professor autograder"
  display_name    = "professor-service"
  project         = "h4i-applications"
}

resource "google_service_account" "queue_invoker" {
  account_id      = "queue-invoker"
  deletion_policy = "ABANDON"
  description     = "Service account used by Cloud Tasks to invoke Cloud Run services"
  display_name    = "Cloud Tasks Queue Invoker"
  project         = "h4i-applications"
}

resource "google_artifact_registry_repository" "professor" {
  cleanup_policy_dry_run = false
  deletion_policy        = "ABANDON"
  description            = "Docker images for h4i-umd autograder"
  format                 = "DOCKER"
  location               = "us-east4"
  mode                   = "STANDARD_REPOSITORY"
  project                = "h4i-applications"
  repository_id          = "professor-repo"

  cleanup_policies {
    action = "DELETE"
    id     = "Delete old versions"

    condition {
      tag_state = "ANY"
    }
  }

  cleanup_policies {
    action = "KEEP"
    id     = "Professor Image Cleanup"

    most_recent_versions {
      keep_count = 2
    }
  }

  vulnerability_scanning_config {
    enablement_config = "DISABLED"
  }
}

resource "google_cloud_tasks_queue" "professor_grading_requests" {
  deletion_policy = "ABANDON"
  desired_state   = "RUNNING"
  location        = "us-east4"
  name            = "professor-grading-requests"
  project         = "h4i-applications"

  rate_limits {
    max_concurrent_dispatches = 10
    max_dispatches_per_second = 500
  }

  retry_config {
    max_attempts  = 5
    max_backoff   = "3600s"
    max_doublings = 16
    min_backoff   = "0.100s"
  }
}

resource "google_cloud_run_v2_service" "professor" {
  deletion_policy     = "ABANDON"
  deletion_protection = true
  ingress             = "INGRESS_TRAFFIC_ALL"
  location            = "us-east4"
  name                = "professor-service"
  project             = "h4i-applications"

  scaling {
    max_instance_count = 10
  }

  template {
    max_instance_request_concurrency = 1
    service_account                  = google_service_account.professor_service.email
    timeout                          = "900s"

    containers {
      # Cloud Run requires an image in the service template, but GitHub Actions owns
      # image updates. The lifecycle block below prevents Terraform from changing this field.
      image = "us-east4-docker.pkg.dev/h4i-applications/professor-repo/professor-service:57d92c0f3b9c477e735d895191f04622a6cf6524"
      name  = "professor-service-1"

      env {
        name = "GITHUB_PAT"

        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.professor_github_pat.secret_id
            version = "latest"
          }
        }
      }

      env {
        name  = "PROJECT_ID"
        value = "h4i-applications"
      }

      ports {
        container_port = 8000
        name           = "http1"
      }

      resources {
        cpu_idle = true
        limits = {
          cpu    = "2"
          memory = "4Gi"
        }
        startup_cpu_boost = true
      }

      startup_probe {
        failure_threshold = 1
        period_seconds    = 240
        timeout_seconds   = 240

        tcp_socket {
          port = 8000
        }
      }
    }

    scaling {
      max_instance_count = 10
    }
  }

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }

  # GitHub Actions updates the image and revision labels on each deploy; gcloud also
  # updates client metadata. Terraform owns the stable service configuration only.
  lifecycle {
    ignore_changes = [
      client,
      client_version,
      template[0].containers[0].image,
      template[0].labels,              
    ]
  }
}

resource "google_iam_workload_identity_pool" "github" {
  deletion_policy           = "ABANDON"
  display_name              = "GitHub Actions Pool"
  project                   = "h4i-applications"
  workload_identity_pool_id = "github"
}

resource "google_iam_workload_identity_pool_provider" "github" {
  attribute_condition = "assertion.repository_owner == 'Hack4Impact-UMD'"
  attribute_mapping = {
    "attribute.actor"            = "assertion.actor"
    "attribute.repository"       = "assertion.repository"
    "attribute.repository_owner" = "assertion.repository_owner"
    "google.subject"             = "assertion.sub"
  }
  deletion_policy                    = "ABANDON"
  display_name                       = "My GitHub repo Provider"
  project                            = "h4i-applications"
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "my-repo"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account_iam_member" "github_actions_wif_user" {
  service_account_id = "projects/h4i-applications/serviceAccounts/github-actions@h4i-applications.iam.gserviceaccount.com"
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/projects/361966602736/locations/global/workloadIdentityPools/github/attribute.repository/Hack4Impact-UMD/professor"
}

resource "google_service_account_iam_member" "github_actions_token_creator" {
  service_account_id = "projects/h4i-applications/serviceAccounts/github-actions@h4i-applications.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "principalSet://iam.googleapis.com/projects/361966602736/locations/global/workloadIdentityPools/github/attribute.repository/Hack4Impact-UMD/professor"
}

resource "google_project_iam_member" "github_actions_artifact_registry_writer" {
  project = "h4i-applications"
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:github-actions@h4i-applications.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_run_admin" {
  project = "h4i-applications"
  role    = "roles/run.admin"
  member  = "serviceAccount:github-actions@h4i-applications.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_service_account_user" {
  project = "h4i-applications"
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:github-actions@h4i-applications.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "professor_service_datastore_user" {
  project = "h4i-applications"
  role    = "roles/datastore.user"
  member  = "serviceAccount:professor-service@h4i-applications.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "professor_service_secret_accessor" {
  project = "h4i-applications"
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:professor-service@h4i-applications.iam.gserviceaccount.com"
}

resource "google_cloud_run_service_iam_member" "professor_queue_invoker" {
  project  = "h4i-applications"
  location = "us-east4"
  service  = "professor-service"
  role     = "roles/run.invoker"
  member   = "serviceAccount:queue-invoker@h4i-applications.iam.gserviceaccount.com"
}

resource "google_monitoring_alert_policy" "professor_queue_depth" {
  combiner              = "OR"
  deletion_policy       = "ABANDON"
  display_name          = "Professor Queue Depth"
  enabled               = true
  notification_channels = local.professor_alert_notification_channels
  project               = "h4i-applications"
  severity              = "WARNING"

  alert_strategy {
    notification_prompts = ["OPENED"]
  }

  conditions {
    display_name = "Cloud Tasks Queue - Queue depth"

    condition_threshold {
      comparison      = "COMPARISON_GT"
      duration        = "0s"
      filter          = "resource.type = \"cloud_tasks_queue\" AND metric.type = \"cloudtasks.googleapis.com/queue/depth\""
      threshold_value = 15

      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_MEAN"
      }

      forecast_options {
        forecast_horizon = "3600s"
      }

      trigger {
        count = 1
      }
    }
  }

  documentation {
    subject = "High Professor Queue Depth"
  }
}

resource "google_monitoring_alert_policy" "professor_queue_high_request_rate" {
  combiner              = "OR"
  deletion_policy       = "ABANDON"
  display_name          = "Professor Queue - High request rate"
  enabled               = true
  notification_channels = local.professor_alert_notification_channels
  project               = "h4i-applications"
  severity              = "WARNING"

  alert_strategy {
    notification_prompts = ["OPENED"]
  }

  conditions {
    display_name = "Cloud Tasks Queue - API requests"

    condition_threshold {
      comparison      = "COMPARISON_GT"
      duration        = "0s"
      filter          = "resource.type = \"cloud_tasks_queue\" AND metric.type = \"cloudtasks.googleapis.com/api/request_count\""
      threshold_value = 5

      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_RATE"
      }

      trigger {
        count = 1
      }
    }
  }
}

resource "google_monitoring_alert_policy" "professor_service_high_memory" {
  combiner              = "OR"
  deletion_policy       = "ABANDON"
  display_name          = "Professor Service - High memory utilization"
  enabled               = true
  notification_channels = local.professor_alert_notification_channels
  project               = "h4i-applications"
  severity              = "WARNING"

  alert_strategy {
    notification_prompts = ["OPENED"]
  }

  conditions {
    display_name = "Cloud Run - Memory utilization"

    condition_threshold {
      comparison      = "COMPARISON_GT"
      duration        = "300s"
      filter          = "resource.type = \"cloud_run_revision\" AND resource.labels.service_name = \"professor-service\" AND metric.type = \"run.googleapis.com/container/memory/utilizations\""
      threshold_value = 0.8

      aggregations {
        alignment_period     = "60s"
        cross_series_reducer = "REDUCE_MEAN"
        group_by_fields      = ["resource.label.service_name"]
        per_series_aligner   = "ALIGN_PERCENTILE_99"
      }

      trigger {
        count = 1
      }
    }
  }

  documentation {
    subject = "High memory utilization on professor-service"
  }
}

resource "google_monitoring_alert_policy" "professor_service_high_cpu" {
  combiner              = "OR"
  deletion_policy       = "ABANDON"
  display_name          = "Professor Service - High CPU utilization"
  enabled               = true
  notification_channels = local.professor_alert_notification_channels
  project               = "h4i-applications"
  severity              = "WARNING"

  alert_strategy {
    notification_prompts = ["OPENED"]
  }

  conditions {
    display_name = "Cloud Run - CPU utilization"

    condition_threshold {
      comparison      = "COMPARISON_GT"
      duration        = "300s"
      filter          = "resource.type = \"cloud_run_revision\" AND resource.labels.service_name = \"professor-service\" AND metric.type = \"run.googleapis.com/container/cpu/utilizations\""
      threshold_value = 0.8

      aggregations {
        alignment_period     = "60s"
        cross_series_reducer = "REDUCE_MEAN"
        group_by_fields      = ["resource.label.service_name"]
        per_series_aligner   = "ALIGN_PERCENTILE_99"
      }

      trigger {
        count = 1
      }
    }
  }

  documentation {
    subject = "High CPU utilization on professor-service"
  }
}

resource "google_monitoring_alert_policy" "professor_service_instance_count" {
  combiner              = "OR"
  deletion_policy       = "ABANDON"
  display_name          = "Professor Service - Instance count above 5"
  enabled               = true
  notification_channels = local.professor_alert_notification_channels
  project               = "h4i-applications"
  severity              = "WARNING"

  alert_strategy {
    notification_prompts = ["OPENED"]
  }

  conditions {
    display_name = "Cloud Run - Instance count"

    condition_threshold {
      comparison      = "COMPARISON_GT"
      duration        = "300s"
      filter          = "resource.type = \"cloud_run_revision\" AND resource.labels.service_name = \"professor-service\" AND metric.type = \"run.googleapis.com/container/instance_count\""
      threshold_value = 5

      aggregations {
        alignment_period     = "60s"
        cross_series_reducer = "REDUCE_SUM"
        group_by_fields      = ["resource.label.service_name"]
        per_series_aligner   = "ALIGN_MEAN"
      }

      trigger {
        count = 1
      }
    }
  }

  documentation {
    subject = "Cloud Run instance count above 5 on professor-service"
  }
}
