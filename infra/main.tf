terraform {
    backend "gcs" {
        bucket = "h4i-applications-terraform-state"
        prefix = "professor/prod"
    }
}

resource "google_secret_manager_secret" "professor_github_pat" {
  project             = "h4i-applications"
  secret_id           = "PROFESSOR_GITHUB_PAT"
  replication {
    auto {
    }
  }
}
