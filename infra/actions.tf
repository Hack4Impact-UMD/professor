locals {
  github_actions_repo_principal = "principalSet://iam.googleapis.com/projects/${local.project_number}/locations/global/workloadIdentityPools/github/attribute.repository/${local.github_repository_owner}/${local.github_repository_name}"
}

resource "google_service_account" "github_actions" {
  account_id      = "github-actions"
  deletion_policy = "ABANDON"
  display_name    = "GitHub Actions"
  project         = local.project_id
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
  service_account_id = "projects/${local.project_id}/serviceAccounts/${google_service_account.github_actions.email}"
  role               = "roles/iam.workloadIdentityUser"
  member             = local.github_actions_repo_principal
}

resource "google_service_account_iam_member" "github_actions_token_creator" {
  service_account_id = "projects/${local.project_id}/serviceAccounts/${google_service_account.github_actions.email}"
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = local.github_actions_repo_principal
}

resource "google_project_iam_member" "github_actions_artifact_registry_writer" {
  project = local.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.github_actions.email}"
}

resource "google_cloud_run_v2_service_iam_member" "github_actions_run_admin" {
  project  = local.project_id
  location = local.region
  name     = google_cloud_run_v2_service.professor.name
  role    = "roles/run.admin"
  member  = "serviceAccount:${google_service_account.github_actions.email}"
}

resource "google_service_account_iam_member" "github_actions_service_account_user" {
  service_account_id = google_service_account.professor_service.name
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${google_service_account.github_actions.email}"
}
