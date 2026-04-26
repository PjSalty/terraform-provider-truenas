package fwresource_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/fwresource"
)

func TestConfigureClient_Nil(t *testing.T) {
	t.Parallel()
	req := resource.ConfigureRequest{}
	resp := &resource.ConfigureResponse{}
	c, ok := fwresource.ConfigureClient(req, resp)
	if ok {
		t.Fatalf("nil ProviderData must return ok=false")
	}
	if c != nil {
		t.Fatalf("nil ProviderData must return nil client, got %v", c)
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("nil ProviderData must not produce a diagnostic, got %v", resp.Diagnostics)
	}
}

func TestConfigureClient_OK(t *testing.T) {
	t.Parallel()
	cli, err := client.NewWithOptions("https://example.invalid", "key", true)
	if err != nil {
		t.Fatalf("build client: %v", err)
	}
	req := resource.ConfigureRequest{ProviderData: cli}
	resp := &resource.ConfigureResponse{}
	got, ok := fwresource.ConfigureClient(req, resp)
	if !ok {
		t.Fatalf("valid ProviderData must return ok=true, diags=%v", resp.Diagnostics)
	}
	if got != cli {
		t.Fatalf("want same *client.Client, got %v", got)
	}
}

func TestConfigureClient_WrongType(t *testing.T) {
	t.Parallel()
	req := resource.ConfigureRequest{ProviderData: "not a client"}
	resp := &resource.ConfigureResponse{}
	got, ok := fwresource.ConfigureClient(req, resp)
	if ok {
		t.Fatalf("wrong-type ProviderData must return ok=false")
	}
	if got != nil {
		t.Fatalf("wrong-type ProviderData must return nil client, got %v", got)
	}
	if !resp.Diagnostics.HasError() {
		t.Fatalf("wrong-type ProviderData must produce a diagnostic")
	}
	if resp.Diagnostics[0].Summary() != "Unexpected Resource Configure Type" {
		t.Fatalf("unexpected diag summary: %q", resp.Diagnostics[0].Summary())
	}
}
