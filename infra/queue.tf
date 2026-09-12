resource "google_service_account" "queue_invoker" {
  account_id      = "queue-invoker"
  deletion_policy = "ABANDON"
  description     = "Service account used by Cloud Tasks to invoke Cloud Run services"
  display_name    = "Cloud Tasks Queue Invoker"
  project         = "h4i-applications"
}

resource "google_cloud_run_service_iam_member" "professor_queue_invoker" {
  project  = "h4i-applications"
  location = "us-east4"
  service  = "professor-service"
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.queue_invoker.email}"
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