package types

// Disk represents a physical disk known to TrueNAS.
//
// Disks are read-only; the API does not expose disk creation or
// deletion (those are done by the underlying OS or by adding
// hardware). The provider exposes Disk only as a data source.
type Disk struct {
	Identifier  string  `json:"identifier"`
	Name        string  `json:"name"`
	Subsystem   string  `json:"subsystem"`
	Number      int     `json:"number"`
	Serial      string  `json:"serial"`
	Size        int64   `json:"size"`
	Description string  `json:"description"`
	Model       string  `json:"model"`
	Type        string  `json:"type"`
	ZFSGuid     *string `json:"zfs_guid"`
	Bus         string  `json:"bus"`
	Devname     string  `json:"devname"`
	Pool        *string `json:"pool"`
}
