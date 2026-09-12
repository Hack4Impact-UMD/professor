resource "google_service_account" "professor_service" {
  account_id      = "professor-service"
  deletion_policy = "ABANDON"
  description     = "Role for Professor autograder"
  display_name    = "professor-service"
  project         = "h4i-applications"
}

resource "google_project_iam_member" "professor_service_datastore_user" { // firestore permission 
  project = "h4i-applications"
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.professor_service.email}"
}

resource "google_project_iam_member" "professor_service_secret_accessor" {
  project = "h4i-applications"
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.professor_service.email}"
}

resource "google_secret_manager_secret" "professor_github_pat" {
  project             = "h4i-applications"
  secret_id           = "PROFESSOR_GITHUB_PAT"
  replication {
    auto {
    }
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
