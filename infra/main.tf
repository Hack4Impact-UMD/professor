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
