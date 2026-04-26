---
page_title: "truenas_cloudsync_credential Resource - terraform-provider-truenas"
subcategory: "Auth & Integration"
description: |-
  Manages a cloud sync credential (S3, B2, Azure, GCS, Dropbox, etc.) on TrueNAS SCALE. Cloud sync credentials live under /cloudsync/credentials and are distinct from keychain credentials (SSH). They are referenced by numeric ID from truenas_cloud_sync and truenas_cloud_backup resources.
---

# truenas_cloudsync_credential (Resource)

Manages a cloud sync credential (S3, B2, Azure, GCS, Dropbox, etc.) on TrueNAS SCALE. Cloud sync credentials live under /cloudsync/credentials and are distinct from keychain credentials (SSH). They are referenced by numeric ID from truenas_cloud_sync and truenas_cloud_backup resources.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
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
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The display name of the credential.
* `provider_type` - (Required) The cloud provider type. One of: S3, B2, AZUREBLOB, GOOGLE_CLOUD_STORAGE, DROPBOX, FTP, SFTP, HTTP, MEGA, OPENSTACK_SWIFT, PCLOUD, WEBDAV, YANDEX, ONEDRIVE, GOOGLE_DRIVE, BACKBLAZE_B2. Changing this forces replacement. Valid values: `S3`, `B2`, `AZUREBLOB`, `GOOGLE_CLOUD_STORAGE`, `DROPBOX`, `FTP`, `SFTP`, `HTTP`, `MEGA`, `OPENSTACK_SWIFT`, `PCLOUD`, `WEBDAV`, `YANDEX`, `ONEDRIVE`, `GOOGLE_DRIVE`, `BACKBLAZE_B2`. Changing this attribute forces a new resource to be created.
* `provider_attributes_json` - (Required) Provider-specific credential fields as a JSON object (e.g. jsonencode({access_key_id = "X", secret_access_key = "Y"})). The exact keys depend on provider_type. Marked sensitive.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_cloudsync_credential` resource.

## Import

The `truenas_cloudsync_credential` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
# Import an existing TrueNAS cloud-sync credential by its numeric ID.
terraform import truenas_cloudsync_credential.example 1
```
