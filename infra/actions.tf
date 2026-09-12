locals { 
    github_actions_service_account_email = "github-actions@h4i-applications.iam.gserviceaccount.com"
    github_actions_repo_principal = "principalSet://iam.googleapis.com/projects/361966602736/locations/global/workloadIdentityPools/github/attribute.repository/Hack4Impact-UMD/professor"
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
  service_account_id = "projects/h4i-applications/serviceAccounts/${local.github_actions_service_account_email}"
  role               = "roles/iam.workloadIdentityUser"
  member             = local.github_actions_repo_principal
}

resource "google_service_account_iam_member" "github_actions_token_creator" {
  service_account_id = "projects/h4i-applications/serviceAccounts/${local.github_actions_service_account_email}"
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = local.github_actions_repo_principal
}

resource "google_project_iam_member" "github_actions_artifact_registry_writer" {
  project = "h4i-applications"
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${local.github_actions_service_account_email}"
}

resource "google_project_iam_member" "github_actions_run_admin" {
  project = "h4i-applications"
  role    = "roles/run.admin"
  member  = "serviceAccount:${local.github_actions_service_account_email}"
}

resource "google_project_iam_member" "github_actions_service_account_user" {
  project = "h4i-applications"
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${local.github_actions_service_account_email}"
}
