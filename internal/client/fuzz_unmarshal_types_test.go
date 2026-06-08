package client_test

import (
	"encoding/json"
	"testing"

	c "github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func FuzzACLEntry_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.ACLEntry
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.ACLEntry
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzACLPerms_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.ACLPerms
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.ACLPerms
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzACMEDNSAuthenticator_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.ACMEDNSAuthenticator
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.ACMEDNSAuthenticator
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzAPIError_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.APIError
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.APIError
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzAPIKey_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.APIKey
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.APIKey
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzAlertClassEntry_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.AlertClassEntry
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.AlertClassEntry
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzAlertClassesConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.AlertClassesConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.AlertClassesConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzAlertService_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.AlertService
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.AlertService
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzApp_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.App
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.App
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzCatalog_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.Catalog
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.Catalog
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzCertificate_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.Certificate
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.Certificate
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzCloudBackup_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.CloudBackup
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.CloudBackup
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzCloudBackupSchedule_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.CloudBackupSchedule
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.CloudBackupSchedule
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzCloudSync_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.CloudSync
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.CloudSync
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzCloudSyncCredential_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.CloudSyncCredential
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.CloudSyncCredential
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzCronJob_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.CronJob
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.CronJob
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzDataset_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.Dataset
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.Dataset
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzDirectoryServicesConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.DirectoryServicesConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.DirectoryServicesConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzDisk_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.Disk
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.Disk
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzFTPConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.FTPConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.FTPConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzFilesystemACL_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.FilesystemACL
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.FilesystemACL
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzFilesystemACLTemplate_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.FilesystemACLTemplate
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.FilesystemACLTemplate
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzFullNetworkConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.FullNetworkConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.FullNetworkConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzGroup_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.Group
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.Group
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSIAuth_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.ISCSIAuth
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.ISCSIAuth
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSIExtent_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.ISCSIExtent
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.ISCSIExtent
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSIInitiator_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.ISCSIInitiator
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.ISCSIInitiator
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSIPortal_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.ISCSIPortal
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.ISCSIPortal
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSIPortalListen_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.ISCSIPortalListen
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.ISCSIPortalListen
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSITarget_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.ISCSITarget
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.ISCSITarget
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSITargetExtent_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.ISCSITargetExtent
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.ISCSITargetExtent
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzISCSITargetGroup_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.ISCSITargetGroup
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.ISCSITargetGroup
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzInitScript_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.InitScript
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.InitScript
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzJob_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.Job
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.Job
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzJobProgress_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.JobProgress
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.JobProgress
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzKMIPConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.KMIPConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.KMIPConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzKerberosKeytab_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.KerberosKeytab
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.KerberosKeytab
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzKerberosRealm_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.KerberosRealm
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.KerberosRealm
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzKeychainCredential_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.KeychainCredential
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.KeychainCredential
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzMailConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.MailConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.MailConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNFSConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NFSConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NFSConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNFSShare_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NFSShare
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NFSShare
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetGlobal_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NVMetGlobal
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NVMetGlobal
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetHost_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NVMetHost
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NVMetHost
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetHostSubsys_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NVMetHostSubsys
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NVMetHostSubsys
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetHostSubsysHost_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NVMetHostSubsysHost
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NVMetHostSubsysHost
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetHostSubsysSubsys_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NVMetHostSubsysSubsys
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NVMetHostSubsysSubsys
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetNamespace_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NVMetNamespace
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NVMetNamespace
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetNamespaceSubsys_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NVMetNamespaceSubsys
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NVMetNamespaceSubsys
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetPort_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NVMetPort
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NVMetPort
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetPortSubsys_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NVMetPortSubsys
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NVMetPortSubsys
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetPortSubsysPort_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NVMetPortSubsysPort
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NVMetPortSubsysPort
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetPortSubsysSubsys_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NVMetPortSubsysSubsys
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NVMetPortSubsysSubsys
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNVMetSubsys_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NVMetSubsys
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NVMetSubsys
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNetworkConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NetworkConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NetworkConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNetworkInterface_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NetworkInterface
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NetworkInterface
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzNetworkInterfaceAlias_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.NetworkInterfaceAlias
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.NetworkInterfaceAlias
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzPool_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.Pool
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.Pool
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzPrivilege_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.Privilege
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.Privilege
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzPrivilegeGroup_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.PrivilegeGroup
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.PrivilegeGroup
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzPropertyRawVal_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.PropertyRawVal
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.PropertyRawVal
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzPropertyValue_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.PropertyValue
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.PropertyValue
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzReplication_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.Replication
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.Replication
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzReportingExporter_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.ReportingExporter
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.ReportingExporter
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzRetryPolicy_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.RetryPolicy
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.RetryPolicy
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzRsyncTask_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.RsyncTask
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.RsyncTask
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSMBConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.SMBConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.SMBConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSMBShare_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.SMBShare
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.SMBShare
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSNMPConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.SNMPConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.SNMPConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSSHConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.SSHConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.SSHConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSchedule_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.Schedule
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.Schedule
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzScrubTask_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.ScrubTask
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.ScrubTask
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzService_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.Service
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.Service
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSetACLEntry_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.SetACLEntry
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.SetACLEntry
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSnapshotTask_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.SnapshotTask
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.SnapshotTask
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzStaticRoute_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.StaticRoute
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.StaticRoute
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSystemDataset_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.SystemDataset
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.SystemDataset
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzSystemInfo_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.SystemInfo
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.SystemInfo
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzTunable_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.Tunable
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.Tunable
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzUPSConfig_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.UPSConfig
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.UPSConfig
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzUpdateCheckResult_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.UpdateCheckResult
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.UpdateCheckResult
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzUpdateTrainInfo_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.UpdateTrainInfo
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.UpdateTrainInfo
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzUpdateTrains_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.UpdateTrains
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.UpdateTrains
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzUser_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.User
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.User
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzUserGroup_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.UserGroup
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.UserGroup
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzVM_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.VM
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.VM
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzVMDevice_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.VMDevice
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.VMDevice
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzVMStatus_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.VMStatus
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.VMStatus
		_ = json.Unmarshal(b, &v2)
	})
}

func FuzzVMware_Unmarshal(f *testing.F) {
	addCommonSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var v c.VMware
		if err := json.Unmarshal(data, &v); err != nil {
			return
		}
		b, err := json.Marshal(&v)
		if err != nil {
			return
		}
		var v2 c.VMware
		_ = json.Unmarshal(b, &v2)
	})
}

