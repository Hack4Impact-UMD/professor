resource "google_service_account" "professor_service" {
  account_id      = local.professor_service_account_id
  deletion_policy = "ABANDON"
  description     = "Role for Professor autograder"
  display_name    = local.professor_service_name
  project         = local.project_id
}

resource "google_project_iam_member" "professor_service_datastore_user" { // firestore permission 
  project = local.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.professor_service.email}"
}

resource "google_secret_manager_secret" "professor_github_pat" {
  project   = local.project_id
  secret_id = "PROFESSOR_GITHUB_PAT"
  replication {
    auto {
    }
  }
}

resource "google_secret_manager_secret_iam_member" "professor_service_github_pat_accessor" {
  project = local.project_id
  secret_id = google_secret_manager_secret.professor_github_pat.secret_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.professor_service.email}"
}

resource "google_cloud_run_v2_service" "professor" {
  deletion_policy     = "ABANDON"
  deletion_protection = true
  ingress             = "INGRESS_TRAFFIC_ALL"
  location            = local.region
  name                = local.professor_service_name
  project             = local.project_id

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
      image = "${local.region}-docker.pkg.dev/${local.project_id}/${local.artifact_repository_id}/${local.professor_service_name}:57d92c0f3b9c477e735d895191f04622a6cf6524"
      name  = "${local.professor_service_name}-1"

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
        value = local.project_id
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
