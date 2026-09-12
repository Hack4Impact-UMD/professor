locals {
  project_id     = "h4i-applications"
  project_number = "361966602736"
  region         = "us-east4"

  github_repository_owner = "Hack4Impact-UMD"
  github_repository_name  = "professor"

  professor_service_name       = "professor-service"
  professor_service_account_id = "professor-service"

  artifact_repository_id = "professor-repo"

  grading_queue_name       = "professor-grading-requests"
  queue_invoker_account_id = "queue-invoker"
}

terraform {
  backend "gcs" {
    bucket = "h4i-applications-terraform-state"
    prefix = "professor/prod"
  }
}

resource "google_artifact_registry_repository" "professor" {
  cleanup_policy_dry_run = false
  deletion_policy        = "ABANDON"
  description            = "Docker images for h4i-umd autograder"
  format                 = "DOCKER"
  location               = local.region
  mode                   = "STANDARD_REPOSITORY"
  project                = local.project_id
  repository_id          = local.artifact_repository_id

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
