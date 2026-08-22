resource "truenas_system_update" "prod" {
  # With autocheck off, TrueNAS never downloads or stages an update on its
  # own. This is the pin: an upgrade needs a deliberate action.
  autocheck = false

  # The update profile this system tracks. Validated against
  # update.profile_choices at apply time, so an unselectable profile is
  # rejected with the valid choices listed rather than failing server-side.
  profile = "MISSION_CRITICAL"
}
