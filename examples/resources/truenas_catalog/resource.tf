# label, location and repository are set by the server. Only the train
# selection and the initial sync are configurable.
resource "truenas_catalog" "example" {
  preferred_trains = ["stable"]
  sync_on_create   = true
}
