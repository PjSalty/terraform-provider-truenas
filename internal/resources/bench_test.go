package resources

// Benchmarks for hot mapping functions: every Read/Create/Update path goes
// through mapResponseToModel at least once, and dataset/VM/certificate/
// cloud_backup are the largest of those models by field count.
//
// Run with:
//
//	go test -run='^$' -bench=. -benchtime=5s ./internal/resources/...
//
// Use -benchtime=1x in CI to keep pipelines fast.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// BenchmarkDatasetMapResponseToModel measures the cost of mapping a
// DatasetResponse (pool, nested name, compression, quota, snapdir, etc.)
// into a DatasetResourceModel. Dataset has the broadest field surface of
// any storage resource.
func BenchmarkDatasetMapResponseToModel(b *testing.B) {
	ds := &client.DatasetResponse{
		ID:            "tank/apps/postgres/data",
		MountPoint:    "/mnt/tank/apps/postgres/data",
		Type:          "FILESYSTEM",
		Compression:   &client.PropertyValue{Value: "LZ4"},
		Atime:         &client.PropertyValue{Value: "OFF"},
		Deduplication: &client.PropertyValue{Value: "OFF"},
		Quota:         &client.PropertyRawVal{Rawvalue: "1073741824", Value: "1073741824"},
		Refquota:      &client.PropertyRawVal{Rawvalue: "536870912", Value: "536870912"},
		Sync:          &client.PropertyValue{Value: "ALWAYS"},
		Snapdir:       &client.PropertyValue{Value: "VISIBLE"},
		Copies:        &client.PropertyValue{Value: "2"},
		Readonly:      &client.PropertyValue{Value: "OFF"},
		RecordSize:    &client.PropertyValue{Value: "128K"},
		ShareType:     &client.PropertyValue{Value: "SMB"},
	}
	r := &DatasetResource{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m DatasetResourceModel
		r.mapResponseToModel(ds, &m)
	}
}

// BenchmarkVMMapResponseToModel measures VMResource.mapResponseToModel —
// ~40 fields, all scalar copies, plus nullable string helpers.
func BenchmarkVMMapResponseToModel(b *testing.B) {
	archType := "x86_64"
	machineType := "q35"
	uuid := "12345678-1234-5678-1234-567812345678"
	cpuModel := "host"
	cpuset := "0-3"
	nodeset := "0"
	minMem := int64(512 * 1024 * 1024)

	vm := &client.VM{
		ID:                    42,
		Name:                  "bench-vm",
		Description:           "benchmark fixture",
		Vcpus:                 1,
		Cores:                 4,
		Threads:               2,
		Memory:                4096 * 1024 * 1024,
		MinMemory:             &minMem,
		Bootloader:            "UEFI",
		BootloaderOvmf:        "OVMF_CODE.fd",
		Autostart:             true,
		HideFromMsr:           false,
		EnsureDisplayDevice:   true,
		Time:                  "LOCAL",
		ShutdownTimeout:       90,
		ArchType:              &archType,
		MachineType:           &machineType,
		UUID:                  &uuid,
		CommandLineArgs:       "",
		CPUMode:               "HOST-PASSTHROUGH",
		CPUModel:              &cpuModel,
		Cpuset:                &cpuset,
		Nodeset:               &nodeset,
		PinVcpus:              false,
		SuspendOnSnapshot:     false,
		TrustedPlatformModule: false,
		HypervEnlightenments:  false,
		EnableSecureBoot:      true,
		Status:                &client.VMStatus{State: "RUNNING"},
	}
	r := &VMResource{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m VMResourceModel
		r.mapResponseToModel(vm, &m)
	}
}

// BenchmarkCertificateMapResponseToModel measures the certificate mapping —
// many optional-string branches plus a SAN list.
func BenchmarkCertificateMapResponseToModel(b *testing.B) {
	cert := &client.Certificate{
		ID:                 7,
		Name:               "bench-cert",
		CertificateData:    "-----BEGIN CERTIFICATE-----\nMIIBenchmark\n-----END CERTIFICATE-----",
		KeyType:            "RSA",
		KeyLength:          4096,
		DigestAlgorithm:    "SHA256",
		Lifetime:           3650,
		DN:                 "CN=bench.example.com",
		From:               "Jan  1 00:00:00 2026 GMT",
		Until:              "Jan  1 00:00:00 2036 GMT",
		Expired:            false,
		Country:            "US",
		State:              "CA",
		City:               "SF",
		Organization:       "Bench Inc.",
		OrganizationalUnit: "IT",
		Email:              "ops@bench.example.com",
		Common:             "bench.example.com",
		SAN:                []string{"bench.example.com", "www.bench.example.com", "api.bench.example.com"},
	}
	r := &CertificateResource{}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m CertificateResourceModel
		r.mapResponseToModel(ctx, cert, &m)
	}
}

// BenchmarkCloudBackupMapResponseToModel measures cloud_backup mapping,
// which exercises JSON canonicalization (attributes_json) in addition to
// scalar field copies.
func BenchmarkCloudBackupMapResponseToModel(b *testing.B) {
	cb := &client.CloudBackup{
		ID:          123,
		Description: "nightly restic",
		Path:        "/mnt/tank/data",
		Credentials: json.RawMessage(`{"id":5,"name":"b2","provider":"BACKBLAZE_B2"}`),
		Attributes: json.RawMessage(
			`{"bucket":"backups","folder":"/nightly","encryption":"aes256","chunk_size":16}`,
		),
		Schedule: client.CloudBackupSchedule{
			Minute: "0", Hour: "2", Dom: "*", Month: "*", Dow: "*",
		},
		PreScript:       "echo start",
		PostScript:      "echo done",
		Snapshot:        true,
		Include:         []string{"/mnt/tank/data/docs", "/mnt/tank/data/photos"},
		Exclude:         []string{"*.tmp", "cache/"},
		Args:            "--verbose",
		Enabled:         true,
		Password:        "",
		KeepLast:        30,
		TransferSetting: "DEFAULT",
	}
	r := &CloudBackupResource{}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m CloudBackupResourceModel
		r.mapResponseToModel(ctx, cb, &m)
	}
}
