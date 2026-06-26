resource "truenas_dataset" "media" {
  pool = "tank"
  name = "media"
}

resource "truenas_directory" "downloaded_music" {
  path           = "${truenas_dataset.media.mount_point}/downloaded/music"
  mode           = "755"
  create_parents = true
  uid            = 1000
  gid            = 1000
}
