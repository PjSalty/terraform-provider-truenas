# Amazon S3 (or any S3-compatible) credential.
resource "truenas_cloudsync_credential" "s3_example" {
  name          = "example-s3"
  provider_type = "S3"
  provider_attributes_json = jsonencode({
    access_key_id     = "AKIAEXAMPLE"
    secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  })
}

# Backblaze B2 credential. Note: TrueNAS uses `B2` (rclone-style) as the
# provider type for Backblaze; `BACKBLAZE_B2` is also accepted as an alias.
resource "truenas_cloudsync_credential" "b2_example" {
  name          = "example-b2"
  provider_type = "B2"
  provider_attributes_json = jsonencode({
    account = "000example000"
    key     = "K000exampleapplicationkey"
  })
}
