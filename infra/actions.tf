locals { 
    github_actions_service_account_email = "github-actions@${local.project_id}.iam.gserviceaccount.com"
    github_actions_repo_principal = "principalSet://iam.googleapis.com/projects/${local.project_number}/locations/global/workloadIdentityPools/github/attribute.repository/${local.github_repository_owner}/${local.github_repository_name}"
}

resource "google_iam_workload_identity_pool" "github" {
  deletion_policy           = "ABANDON"
  display_name              = "GitHub Actions Pool"
  project                   = local.project_id
  workload_identity_pool_id = "github"
}

resource "google_iam_workload_identity_pool_provider" "github" {
  attribute_condition = "assertion.repository_owner == '${local.github_repository_owner}'"
  attribute_mapping = {
    "attribute.actor"            = "assertion.actor"
    "attribute.repository"       = "assertion.repository"
    "attribute.repository_owner" = "assertion.repository_owner"
    "google.subject"             = "assertion.sub"
  }
  deletion_policy                    = "ABANDON"
  display_name                       = "My GitHub repo Provider"
  project                            = local.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "my-repo"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account_iam_member" "github_actions_wif_user" {
  service_account_id = "projects/${local.project_id}/serviceAccounts/${local.github_actions_service_account_email}"
  role               = "roles/iam.workloadIdentityUser"
  member             = local.github_actions_repo_principal
}

resource "google_service_account_iam_member" "github_actions_token_creator" {
  service_account_id = "projects/${local.project_id}/serviceAccounts/${local.github_actions_service_account_email}"
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = local.github_actions_repo_principal
}

resource "google_project_iam_member" "github_actions_artifact_registry_writer" {
  project = local.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${local.github_actions_service_account_email}"
}

resource "google_project_iam_member" "github_actions_run_admin" {
  project = local.project_id
  role    = "roles/run.admin"
  member  = "serviceAccount:${local.github_actions_service_account_email}"
}

resource "google_project_iam_member" "github_actions_service_account_user" {
  project = local.project_id
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${local.github_actions_service_account_email}"
}
