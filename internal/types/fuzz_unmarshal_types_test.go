package types_test

import (
	"encoding/json"
	"testing"

	t "github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func FuzzACLEntry_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.ACLEntry
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.ACLEntry
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzACLPerms_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.ACLPerms
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.ACLPerms
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzACMEDNSAuthenticator_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.ACMEDNSAuthenticator
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.ACMEDNSAuthenticator
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzAPIKey_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.APIKey
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.APIKey
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzAlertClassEntry_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.AlertClassEntry
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.AlertClassEntry
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzAlertClassesConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.AlertClassesConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.AlertClassesConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzAlertService_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.AlertService
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.AlertService
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzApp_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.App
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.App
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzCatalog_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.Catalog
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.Catalog
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzCertificate_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.Certificate
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.Certificate
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzCloudBackup_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.CloudBackup
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.CloudBackup
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzCloudBackupSchedule_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.CloudBackupSchedule
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.CloudBackupSchedule
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzCloudSync_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.CloudSync
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.CloudSync
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzCloudSyncCredential_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.CloudSyncCredential
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.CloudSyncCredential
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzCronJob_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.CronJob
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.CronJob
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzDataset_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.Dataset
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.Dataset
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzDirectoryServicesConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.DirectoryServicesConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.DirectoryServicesConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzDisk_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.Disk
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.Disk
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzFTPConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.FTPConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.FTPConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzFilesystemACL_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.FilesystemACL
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.FilesystemACL
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzFilesystemACLTemplate_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.FilesystemACLTemplate
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.FilesystemACLTemplate
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzFullNetworkConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.FullNetworkConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.FullNetworkConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzGroup_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.Group
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.Group
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSIAuth_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.ISCSIAuth
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.ISCSIAuth
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSIExtent_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.ISCSIExtent
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.ISCSIExtent
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSIInitiator_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.ISCSIInitiator
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.ISCSIInitiator
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSIPortal_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.ISCSIPortal
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.ISCSIPortal
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSIPortalListen_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.ISCSIPortalListen
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.ISCSIPortalListen
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSITarget_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.ISCSITarget
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.ISCSITarget
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSITargetExtent_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.ISCSITargetExtent
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.ISCSITargetExtent
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSITargetGroup_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.ISCSITargetGroup
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.ISCSITargetGroup
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzInitScript_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.InitScript
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.InitScript
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzKMIPConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.KMIPConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.KMIPConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzKerberosKeytab_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.KerberosKeytab
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.KerberosKeytab
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzKerberosRealm_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.KerberosRealm
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.KerberosRealm
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzKeychainCredential_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.KeychainCredential
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.KeychainCredential
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzMailConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.MailConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.MailConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNFSConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NFSConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NFSConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNFSShare_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NFSShare
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NFSShare
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetGlobal_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NVMetGlobal
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NVMetGlobal
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetHost_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NVMetHost
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NVMetHost
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetHostSubsys_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NVMetHostSubsys
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NVMetHostSubsys
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetHostSubsysHost_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NVMetHostSubsysHost
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NVMetHostSubsysHost
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetHostSubsysSubsys_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NVMetHostSubsysSubsys
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NVMetHostSubsysSubsys
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetNamespace_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NVMetNamespace
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NVMetNamespace
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetNamespaceSubsys_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NVMetNamespaceSubsys
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NVMetNamespaceSubsys
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetPort_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NVMetPort
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NVMetPort
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetPortSubsys_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NVMetPortSubsys
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NVMetPortSubsys
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetPortSubsysPort_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NVMetPortSubsysPort
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NVMetPortSubsysPort
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetPortSubsysSubsys_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NVMetPortSubsysSubsys
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NVMetPortSubsysSubsys
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetSubsys_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NVMetSubsys
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NVMetSubsys
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNetworkConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NetworkConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NetworkConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNetworkInterface_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NetworkInterface
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NetworkInterface
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNetworkInterfaceAlias_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.NetworkInterfaceAlias
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.NetworkInterfaceAlias
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzPool_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.Pool
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.Pool
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzPrivilege_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.Privilege
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.Privilege
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzPrivilegeGroup_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.PrivilegeGroup
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.PrivilegeGroup
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzPropertyRawVal_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.PropertyRawVal
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.PropertyRawVal
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzPropertyValue_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.PropertyValue
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.PropertyValue
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzReplication_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.Replication
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.Replication
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzReportingExporter_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.ReportingExporter
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.ReportingExporter
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzRsyncTask_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.RsyncTask
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.RsyncTask
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSMBConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.SMBConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.SMBConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSMBShare_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.SMBShare
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.SMBShare
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSNMPConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.SNMPConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.SNMPConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSSHConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.SSHConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.SSHConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSchedule_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.Schedule
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.Schedule
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzScrubTask_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.ScrubTask
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.ScrubTask
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzService_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.Service
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.Service
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSetACLEntry_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.SetACLEntry
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.SetACLEntry
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSnapshotTask_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.SnapshotTask
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.SnapshotTask
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzStaticRoute_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.StaticRoute
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.StaticRoute
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSystemDataset_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.SystemDataset
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.SystemDataset
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSystemInfo_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.SystemInfo
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.SystemInfo
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzTunable_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.Tunable
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.Tunable
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzUPSConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.UPSConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.UPSConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzUpdateCheckResult_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.UpdateCheckResult
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.UpdateCheckResult
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzUpdateTrainInfo_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.UpdateTrainInfo
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.UpdateTrainInfo
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzUpdateTrains_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.UpdateTrains
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.UpdateTrains
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzUser_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.User
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.User
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzUserGroup_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.UserGroup
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.UserGroup
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzVM_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.VM
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.VM
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzVMDeleteOptions_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.VMDeleteOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.VMDeleteOptions
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzVMDevice_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.VMDevice
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.VMDevice
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzVMStatus_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.VMStatus
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.VMStatus
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzVMware_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v t.VMware
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 t.VMware
		_ = json.Unmarshal(b, &v2)
	})
}

