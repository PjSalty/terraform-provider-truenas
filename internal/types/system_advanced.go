package types

// SystemAdvancedSyslogServer is one remote syslog destination.
//
// Host is a DNS name or IP address, with an optional ":port" suffix. An IPv6
// address is written in brackets, "[2001:db8::10]:514". Without a port
// TrueNAS uses 514 for UDP and TCP and 6514 for TLS. TLSCertificate is the
// certificate sent for mutual TLS and is null for every other case.
type SystemAdvancedSyslogServer struct {
	Host           string `json:"host"`
	Transport      string `json:"transport"`
	TLSCertificate *int64 `json:"tls_certificate"`
}

// SystemAdvanced is the System > Advanced settings singleton, read with
// system.advanced.config and written with system.advanced.update.
//
// Only the attributes the provider manages are decoded. The rest of the
// entry (advancedmode, autotune, consolemsg, sed_user, anonstats_token,
// isolated_gpu_pci_ids) is ignored, see the resource documentation.
type SystemAdvanced struct {
	ID                 int                          `json:"id"`
	Consolemenu        bool                         `json:"consolemenu"`
	Serialconsole      bool                         `json:"serialconsole"`
	Serialport         string                       `json:"serialport"`
	Serialspeed        string                       `json:"serialspeed"`
	Powerdaemon        bool                         `json:"powerdaemon"`
	Overprovision      *int64                       `json:"overprovision"`
	Traceback          bool                         `json:"traceback"`
	Debugkernel        bool                         `json:"debugkernel"`
	Uploadcrash        bool                         `json:"uploadcrash"`
	Anonstats          bool                         `json:"anonstats"`
	Motd               string                       `json:"motd"`
	LoginBanner        string                       `json:"login_banner"`
	BootScrub          int64                        `json:"boot_scrub"`
	FQDNSyslog         bool                         `json:"fqdn_syslog"`
	Sysloglevel        string                       `json:"sysloglevel"`
	Syslogservers      []SystemAdvancedSyslogServer `json:"syslogservers"`
	SyslogAudit        bool                         `json:"syslog_audit"`
	KdumpEnabled       bool                         `json:"kdump_enabled"`
	KernelExtraOptions string                       `json:"kernel_extra_options"`
}

// SystemAdvancedUpdateRequest is the body for system.advanced.update. Every
// field is optional: the upstream model is ForUpdateMetaclass, so an omitted
// key leaves the stored value alone. Syslogservers is a pointer to a slice
// because an empty list is a real value (it removes every remote server)
// that has to be sent, while nil means "leave the servers alone".
type SystemAdvancedUpdateRequest struct {
	Consolemenu        *bool                         `json:"consolemenu,omitempty"`
	Serialconsole      *bool                         `json:"serialconsole,omitempty"`
	Serialport         *string                       `json:"serialport,omitempty"`
	Serialspeed        *string                       `json:"serialspeed,omitempty"`
	Powerdaemon        *bool                         `json:"powerdaemon,omitempty"`
	Overprovision      *int64                        `json:"overprovision,omitempty"`
	Traceback          *bool                         `json:"traceback,omitempty"`
	Debugkernel        *bool                         `json:"debugkernel,omitempty"`
	Uploadcrash        *bool                         `json:"uploadcrash,omitempty"`
	Anonstats          *bool                         `json:"anonstats,omitempty"`
	Motd               *string                       `json:"motd,omitempty"`
	LoginBanner        *string                       `json:"login_banner,omitempty"`
	BootScrub          *int64                        `json:"boot_scrub,omitempty"`
	FQDNSyslog         *bool                         `json:"fqdn_syslog,omitempty"`
	Sysloglevel        *string                       `json:"sysloglevel,omitempty"`
	Syslogservers      *[]SystemAdvancedSyslogServer `json:"syslogservers,omitempty"`
	SyslogAudit        *bool                         `json:"syslog_audit,omitempty"`
	KdumpEnabled       *bool                         `json:"kdump_enabled,omitempty"`
	KernelExtraOptions *string                       `json:"kernel_extra_options,omitempty"`
}
