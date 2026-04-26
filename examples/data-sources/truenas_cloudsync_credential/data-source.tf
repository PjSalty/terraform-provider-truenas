data "truenas_cloudsync_credential" "by_name" {
  name = "example-s3"
}

output "example_s3_id" {
  value = data.truenas_cloudsync_credential.by_name.id
}

output "example_s3_provider_type" {
  value = data.truenas_cloudsync_credential.by_name.provider_type
}
