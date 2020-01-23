output "url" {
  description = "The URL where the Cloud Run Service can be accessed."
  value       = google_cloud_run_service.service.status[0].url
}

output "trigger_id" {
  description = "The unique identifier for the Cloud Build trigger."
  value       = google_cloudbuild_trigger.cloud_build_trigger.trigger_id
}

output "repository_http_url" {
  description = "HTTP URL of the repository in Cloud Source Repositories."
  value       = google_sourcerepo_repository.repo.url
}
