resource "truenas_system_update" "prod" {
  # Leave TrueNAS's automatic daily check enabled so the UI still shows
  # "update available" banners, but refuse to auto-download updates.
  # Disabling auto_download is the primary "pin" lever — TrueNAS will
  # never stage an update without a conscious action.
  auto_download = false

  # Pin the active release train. Change this attribute to migrate to a
  # newer release train; a plan+apply is required to reconcile.
  train = "TrueNAS-SCALE-Fangtooth"
}
