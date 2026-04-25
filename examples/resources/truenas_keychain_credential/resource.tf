resource "truenas_keychain_credential" "example" {
  name = "aws-s3-backup"
  type = "S3_BUCKET"
  attributes = {
    access_key_id     = "AKIAEXAMPLE"
    secret_access_key = "examplesecret"
  }
}
