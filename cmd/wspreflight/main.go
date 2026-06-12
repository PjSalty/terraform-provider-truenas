// Command wspreflight is the acceptance suite's preflight probe.
// It dials the TrueNAS JSON-RPC WebSocket API with the env-supplied
// credentials and prints "<version>\t<pool1,pool2,...>" on success.
//
// The shell preflight (scripts/lib/_env.sh) used to curl the REST
// /api/v2.0/system/info endpoint, which TrueNAS 26 removed. This
// probe travels the same WS path the provider itself uses, so the
// preflight validates exactly what the suite will exercise.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var errUsage = errors.New("TRUENAS_URL and TRUENAS_API_KEY must be set")

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if errors.Is(err, errUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func run() error {
	url := os.Getenv("TRUENAS_URL")
	key := os.Getenv("TRUENAS_API_KEY")
	insecure := os.Getenv("TRUENAS_INSECURE_SKIP_VERIFY") == "true" ||
		os.Getenv("TRUENAS_INSECURE_SKIP_VERIFY") == "1"
	if url == "" || key == "" {
		return errUsage
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := wsclient.New(ctx, url, key, insecure)
	if err != nil {
		return fmt.Errorf("dial/auth: %w", err)
	}

	info, err := c.GetSystemInfo(ctx)
	if err != nil {
		return fmt.Errorf("system.info: %w", err)
	}

	pools, err := c.ListPools(ctx)
	if err != nil {
		return fmt.Errorf("pool.query: %w", err)
	}
	names := make([]string, 0, len(pools))
	for _, p := range pools {
		names = append(names, p.Name)
	}

	// Trim the TrueNAS-SCALE- prefix some versions include.
	version := strings.TrimPrefix(info.Version, "TrueNAS-SCALE-")
	version = strings.TrimPrefix(version, "TrueNAS-")
	out, _ := json.Marshal(map[string]interface{}{
		"version": version,
		"pools":   names,
	})
	fmt.Println(string(out))
	return nil
}
