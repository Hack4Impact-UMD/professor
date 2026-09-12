locals {
  professor_alert_notification_channels = [
    "projects/${local.project_id}/notificationChannels/8006514687676133218",
    "projects/${local.project_id}/notificationChannels/13261586194715419281",
    "projects/${local.project_id}/notificationChannels/12677533325804840933",
  ]
}

resource "google_monitoring_alert_policy" "professor_queue_depth" {
  combiner              = "OR"
  deletion_policy       = "ABANDON"
  display_name          = "Professor Queue Depth"
  enabled               = true
  notification_channels = local.professor_alert_notification_channels
  project               = local.project_id
  severity              = "WARNING"

  alert_strategy {
    notification_prompts = ["OPENED"]
  }

  conditions {
    display_name = "Cloud Tasks Queue - Queue depth"

    condition_threshold {
      comparison      = "COMPARISON_GT"
      duration        = "0s"
      filter          = "resource.type = \"cloud_tasks_queue\" AND metric.type = \"cloudtasks.googleapis.com/queue/depth\""
      threshold_value = 15

      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_MEAN"
      }

      forecast_options {
        forecast_horizon = "3600s"
      }

      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = "The Professor Cloud Tasks queue has more than 15 pending tasks."
    mime_type = "text/markdown"
    subject   = "High Professor Queue Depth"
  }
}

resource "google_monitoring_alert_policy" "professor_queue_high_request_rate" {
  combiner              = "OR"
  deletion_policy       = "ABANDON"
  display_name          = "Professor Queue - High request rate"
  enabled               = true
  notification_channels = local.professor_alert_notification_channels
  project               = local.project_id
  severity              = "WARNING"

  alert_strategy {
    notification_prompts = ["OPENED"]
  }

  conditions {
    display_name = "Cloud Tasks Queue - API requests"

    condition_threshold {
      comparison      = "COMPARISON_GT"
      duration        = "0s"
      filter          = "resource.type = \"cloud_tasks_queue\" AND metric.type = \"cloudtasks.googleapis.com/api/request_count\""
      threshold_value = 5

      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_RATE"
      }

      trigger {
        count = 1
      }
    }
  }
}

resource "google_monitoring_alert_policy" "professor_service_high_memory" {
  combiner              = "OR"
  deletion_policy       = "ABANDON"
  display_name          = "Professor Service - High memory utilization"
  enabled               = true
  notification_channels = local.professor_alert_notification_channels
  project               = local.project_id
  severity              = "WARNING"

  alert_strategy {
    notification_prompts = ["OPENED"]
  }

  conditions {
    display_name = "Cloud Run - Memory utilization"

    condition_threshold {
      comparison      = "COMPARISON_GT"
      duration        = "300s"
      filter          = "resource.type = \"cloud_run_revision\" AND resource.labels.service_name = \"${local.professor_service_name}\" AND metric.type = \"run.googleapis.com/container/memory/utilizations\""
      threshold_value = 0.8

      aggregations {
        alignment_period     = "60s"
        cross_series_reducer = "REDUCE_MEAN"
        group_by_fields      = ["resource.label.service_name"]
        per_series_aligner   = "ALIGN_PERCENTILE_99"
      }

      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = "The Professor Cloud Run service has memory utilization above 80% for 5 minutes."
    mime_type = "text/markdown"
    subject   = "High memory utilization on ${local.professor_service_name}"
  }
}

resource "google_monitoring_alert_policy" "professor_service_high_cpu" {
  combiner              = "OR"
  deletion_policy       = "ABANDON"
  display_name          = "Professor Service - High CPU utilization"
  enabled               = true
  notification_channels = local.professor_alert_notification_channels
  project               = local.project_id
  severity              = "WARNING"

  alert_strategy {
    notification_prompts = ["OPENED"]
  }

  conditions {
    display_name = "Cloud Run - CPU utilization"

    condition_threshold {
      comparison      = "COMPARISON_GT"
      duration        = "300s"
      filter          = "resource.type = \"cloud_run_revision\" AND resource.labels.service_name = \"${local.professor_service_name}\" AND metric.type = \"run.googleapis.com/container/cpu/utilizations\""
      threshold_value = 0.8

      aggregations {
        alignment_period     = "60s"
        cross_series_reducer = "REDUCE_MEAN"
        group_by_fields      = ["resource.label.service_name"]
        per_series_aligner   = "ALIGN_PERCENTILE_99"
      }

      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = "The Professor Cloud Run service has CPU utilization above 80% for 5 minutes."
    mime_type = "text/markdown"
    subject   = "High CPU utilization on ${local.professor_service_name}"
  }
}

resource "google_monitoring_alert_policy" "professor_service_instance_count" {
  combiner              = "OR"
  deletion_policy       = "ABANDON"
  display_name          = "Professor Service - Instance count above 5"
  enabled               = true
  notification_channels = local.professor_alert_notification_channels
  project               = local.project_id
  severity              = "WARNING"

  alert_strategy {
    notification_prompts = ["OPENED"]
  }

  conditions {
    display_name = "Cloud Run - Instance count"

    condition_threshold {
      comparison      = "COMPARISON_GT"
      duration        = "300s"
      filter          = "resource.type = \"cloud_run_revision\" AND resource.labels.service_name = \"${local.professor_service_name}\" AND metric.type = \"run.googleapis.com/container/instance_count\""
      threshold_value = 5

      aggregations {
        alignment_period     = "60s"
        cross_series_reducer = "REDUCE_SUM"
        group_by_fields      = ["resource.label.service_name"]
        per_series_aligner   = "ALIGN_MEAN"
      }

      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = "The Professor Cloud Run service has more than 5 active instances for 5 minutes."
    mime_type = "text/markdown"
    subject   = "Cloud Run instance count above 5 on ${local.professor_service_name}"
  }
}
