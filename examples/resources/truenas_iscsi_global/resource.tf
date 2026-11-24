# iSCSI global configuration: the IQN prefix, the iSNS servers, the listen
# port and the pool space threshold.
#
# This is a singleton: TrueNAS has exactly one. Anything left out keeps the
# value TrueNAS has.

resource "truenas_iscsi_global" "this" {
  # Every target IQN is "<basename>:<target name>".
  basename = "iqn.2005-10.org.freenas.ctl"

  # IP addresses with an optional port (3205 by default), not hostnames.
  # TrueNAS must be able to connect to each one when this is applied.
  isns_servers = ["10.0.0.5"]

  listen_port = 3260

  # Pool available space threshold in percent, 1 to 99. 0 turns it off.
  pool_avail_threshold = 80
}
