# Replication global configuration: how many replication tasks may run at
# the same time.
#
# This is a singleton: TrueNAS has exactly one.

resource "truenas_replication_config" "this" {
  # At least 1. TrueNAS defaults to 5. 0 means no limit.
  max_parallel_replication_tasks = 3
}
