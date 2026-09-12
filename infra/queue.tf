resource "google_service_account" "queue_invoker" {
  account_id      = local.queue_invoker_account_id
  deletion_policy = "ABANDON"
  description     = "Service account used by Cloud Tasks to invoke Cloud Run services"
  display_name    = "Cloud Tasks Queue Invoker"
  project         = local.project_id
}

resource "google_cloud_run_service_iam_member" "professor_queue_invoker" {
  project  = local.project_id
  location = local.region
  service  = local.professor_service_name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.queue_invoker.email}"
}

resource "google_cloud_tasks_queue" "professor_grading_requests" {
  deletion_policy = "ABANDON"
  desired_state   = "RUNNING"
  location        = local.region
  name            = local.grading_queue_name
  project         = local.project_id

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
