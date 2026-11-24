# Pool resilver priority window. Inside the window TrueNAS lets a resilver
# use more of the disks, so a replaced disk is back in the pool sooner.
# Outside it the ZFS defaults apply and client I/O comes first.
#
# This is a singleton: TrueNAS has exactly one window, for every pool.

resource "truenas_resilver_config" "this" {
  enabled = true

  # Zero-padded 24 hour times in the system time zone. A begin later than
  # the end rolls over midnight: this window runs from 18:00 to 09:00.
  begin = "18:00"
  end   = "09:00"

  # 1 is Monday, 7 is Sunday.
  weekday = [1, 2, 3, 4, 5, 6, 7]
}
