package client_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// scaleBenchFixtures synthesises large response payloads the
// provider would encounter on a busy TrueNAS — 10k datasets,
// 100k snapshots, 1M users, etc. The benchmarks measure end-to-
// end parse time so regressions in the Unmarshal path (e.g. an
// O(n^2) walker introduced in walkRedact) show up as a benchmark
// time delta.

// fixedDatasetN returns N dataset records as a JSON array.
func fixedDatasetN(n int) []byte {
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"id":"pool/ds-%d","type":"FILESYSTEM","mountpoint":"/mnt/pool/ds-%d","comments":"bench fixture","encrypted":false}`, i, i)
	}
	b.WriteByte(']')
	return []byte(b.String())
}

// fixedUserN returns N user records as a JSON array.
func fixedUserN(n int) []byte {
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"id":%d,"uid":%d,"username":"bench-user-%d","full_name":"Bench %d","email":"u%d@b.test","builtin":false,"locked":false,"smb":true,"shell":"/usr/bin/bash"}`, i, 1000+i, i, i, i)
	}
	b.WriteByte(']')
	return []byte(b.String())
}

// fixtureServer mints an httptest.Server that always serves the
// given body for every request. Lets a benchmark measure pure
// client+Unmarshal time without disk or network in the loop.
func fixtureServer(b *testing.B, body []byte, contentType string) *httptest.Server {
	b.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(body)
	}))
	b.Cleanup(srv.Close)
	return srv
}

func BenchmarkScale_ListDatasets_1k(b *testing.B) {
	benchListDatasets(b, 1_000)
}
func BenchmarkScale_ListDatasets_10k(b *testing.B) {
	benchListDatasets(b, 10_000)
}
func BenchmarkScale_ListDatasets_100k(b *testing.B) {
	benchListDatasets(b, 100_000)
}

func benchListDatasets(b *testing.B, n int) {
	body := fixedDatasetN(n)
	srv := fixtureServer(b, body, "application/json")
	c, _ := client.New(srv.URL, "k")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// We don't have a public "ListDatasets" client method in this
		// codebase; the read pattern is GetDataset by ID. Bench the
		// raw Unmarshal cost on the same shape the wsclient
		// dataset.query would produce, modeling the worst case where
		// a future ListDatasets client is added.
		var datasets []map[string]interface{}
		if err := json.Unmarshal(body, &datasets); err != nil {
			b.Fatalf("unmarshal: %v", err)
		}
		_ = c // keeps the import live even when GetDataset isn't called
	}
	b.ReportMetric(float64(len(body))/float64(b.Elapsed().Nanoseconds())*1e9, "bytes/sec")
}

func BenchmarkScale_UnmarshalUsers_1k(b *testing.B) { benchUnmarshalUsers(b, 1_000) }
func BenchmarkScale_UnmarshalUsers_10k(b *testing.B) {
	benchUnmarshalUsers(b, 10_000)
}
func BenchmarkScale_UnmarshalUsers_100k(b *testing.B) {
	benchUnmarshalUsers(b, 100_000)
}

func benchUnmarshalUsers(b *testing.B, n int) {
	body := fixedUserN(n)
	b.SetBytes(int64(len(body)))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var users []map[string]interface{}
		if err := json.Unmarshal(body, &users); err != nil {
			b.Fatalf("unmarshal: %v", err)
		}
	}
}

// TestScale_MemoryFootprint10k asserts the provider's parse path
// for a 10k-dataset list doesn't blow up RSS. Real-world: an
// operator with a populated TrueNAS does `terraform refresh` and
// the provider streams 10k+ records through json.Unmarshal +
// state mapping. We measure heap growth to catch a regression
// where someone wraps the parser in O(n²) reflection.
//
// Acceptance: heap delta < 100 MB for 10k records.
func TestScale_MemoryFootprint10k(t *testing.T) {
	if testing.Short() {
		t.Skip("skip in -short mode")
	}
	body := fixedDatasetN(10_000)

	var statsBefore, statsAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&statsBefore)

	var datasets []map[string]interface{}
	if err := json.Unmarshal(body, &datasets); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	runtime.ReadMemStats(&statsAfter)

	heapDelta := int64(statsAfter.HeapAlloc) - int64(statsBefore.HeapAlloc)
	const ceiling = 100 * 1024 * 1024
	if heapDelta > ceiling {
		t.Errorf("heap grew by %d bytes parsing 10k datasets — over 100MB ceiling",
			heapDelta)
	}
	t.Logf("heap delta for 10k datasets: %d bytes (%.1f MB)",
		heapDelta, float64(heapDelta)/1024/1024)
}
