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
