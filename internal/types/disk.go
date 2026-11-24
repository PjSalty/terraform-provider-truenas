package types

// Disk represents a physical disk known to TrueNAS.
//
// The API does not create or delete disks (the OS and the hardware do),
// so the provider reads them through the disk data source and manages only
// the per-disk settings, HDDStandby, AdvPowerMgmt and Description, through
// the truenas_disk resource.
//
// TrueNAS reports HDDStandby and AdvPowerMgmt upper-cased ("ALWAYS ON",
// "DISABLED", or a number as text). Size, Model and Type are null upstream
// for some devices and decode to their zero value here.
type Disk struct {
	Identifier   string  `json:"identifier"`
	Name         string  `json:"name"`
	Subsystem    string  `json:"subsystem"`
	Number       int     `json:"number"`
	Serial       string  `json:"serial"`
	Size         int64   `json:"size"`
	Description  string  `json:"description"`
	HDDStandby   string  `json:"hddstandby"`
	AdvPowerMgmt string  `json:"advpowermgmt"`
	Model        string  `json:"model"`
	Type         string  `json:"type"`
	ZFSGuid      *string `json:"zfs_guid"`
	Bus          string  `json:"bus"`
	Devname      string  `json:"devname"`
	Pool         *string `json:"pool"`
}

// DiskUpdateRequest is the body for disk.update. Every field is optional:
// the upstream model is ForUpdateMetaclass, so an omitted key leaves the
// stored value alone. Only the settings the provider manages are here; the
// other keys the model still accepts (bus, number, lunid, pool, enclosure,
// passwd) describe the hardware or hold a secret and are never sent.
type DiskUpdateRequest struct {
	HDDStandby   *string `json:"hddstandby,omitempty"`
	AdvPowerMgmt *string `json:"advpowermgmt,omitempty"`
	Description  *string `json:"description,omitempty"`
}
