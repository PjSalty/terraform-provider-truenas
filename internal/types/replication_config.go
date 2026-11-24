package types

// ReplicationConfig is the replication global configuration.
//
// It is a ConfigService singleton: one row, read with
// replication.config.config and written with replication.config.update.
//
// MaxParallelReplicationTasks is null when there is no limit.
type ReplicationConfig struct {
	ID                          int  `json:"id"`
	MaxParallelReplicationTasks *int `json:"max_parallel_replication_tasks"`
}

// ReplicationConfigUpdateRequest is the body for replication.config.update.
// The upstream model is ForUpdateMetaclass, so an omitted key leaves the
// stored value alone. A Nullable leaves the key out, sends a value, or sends
// null, which removes the limit.
type ReplicationConfigUpdateRequest struct {
	MaxParallelReplicationTasks Nullable[int] `json:"max_parallel_replication_tasks,omitzero"`
}
