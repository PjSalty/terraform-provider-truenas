# Makefile for terraform-provider-truenas
#
# Standard HashiCorp-style provider build targets. Prefer this Makefile for
# local development. CI uses the same targets via .gitlab-ci.yml.

BINARY       := terraform-provider-truenas
VERSION      ?= 0.1.0
NAMESPACE    := local/saltstice
PLUGIN_DIR   := $(HOME)/.terraform.d/plugins/$(NAMESPACE)/truenas/$(VERSION)/linux_amd64
GO           ?= go
GOFMT        ?= gofmt
# golangci-lint is a dev-time tool installed via `go install` (lands in
# $GOPATH/bin ~ $HOME/go/bin). Prefer whatever is on PATH; fall back to
# $GOPATH/bin/golangci-lint so `make lint` / `make prod-ready` work out
# of the box on a fresh checkout where $GOPATH/bin isn't exported.
GOLANGCI_LINT ?= $(shell command -v golangci-lint 2>/dev/null || echo $(shell go env GOPATH)/bin/golangci-lint)
TFPLUGINDOCS ?= tfplugindocs

PKGS         := ./...
INTERNAL     := ./internal/...

.DEFAULT_GOAL := default

.PHONY: default build install test testacc fmt fmtcheck vet lint tidy docs clean help prod-ready

default: build

## build: Compile the provider binary into the repo root.
build:
	$(GO) build -o $(BINARY)

## install: Build and install the provider into ~/.terraform.d/plugins for local testing.
install: build
	mkdir -p $(PLUGIN_DIR)
	cp $(BINARY) $(PLUGIN_DIR)/$(BINARY)_v$(VERSION)

## test: Run unit tests (httptest-mocked, no live infrastructure).
test:
	$(GO) test -v -count=1 -race -coverprofile=coverage.out $(INTERNAL)

## prod-ready: Run the Phase B battle-hardening invariant tests that gate
## a safe production rollout. Fast (<5s), no live infrastructure, no TF_ACC.
## Run this before any tag that you intend to point at a real TrueNAS.
prod-ready:
	@echo "==> Sweeper coverage invariant"
	$(GO) test -count=1 -run '^TestSweeperCoverage$$' ./internal/provider/
	@echo "==> Delete IsNotFound invariant"
	$(GO) test -count=1 -run '^TestDeleteHandlesNotFound$$' ./internal/provider/
	@echo "==> CRUD logging invariant"
	$(GO) test -count=1 -run '^TestCRUDLogging$$' ./internal/provider/
	@echo "==> State persistence invariant"
	$(GO) test -count=1 -run '^TestStatePersistence$$' ./internal/provider/
	@echo "==> Timeouts block invariant"
	$(GO) test -count=1 -run '^TestResourcesHaveTimeoutsBlock$$' ./internal/provider/
	@echo "==> Apply-idempotency coverage ratchet"
	$(GO) test -count=1 -run '^TestIdempotencyCheckCoverage$$' ./internal/provider/
	@echo "==> Read-only safety rail"
	$(GO) test -count=1 -run '^TestReadOnly_' ./internal/client/
	$(GO) test -count=1 -run '^TestProvider_Configure_ReadOnly' ./internal/provider/
	$(GO) test -count=1 -run '^TestIntegration_ReadOnly' ./internal/provider/
	@echo "==> Fault injection"
	$(GO) test -count=1 -run '^TestFault_' ./internal/client/
	@echo "==> Request timeout plumbing"
	$(GO) test -count=1 -run '^TestRequestTimeout_' ./internal/client/
	$(GO) test -count=1 -run '^TestProvider_Configure_RequestTimeout' ./internal/provider/
	@echo "==> Phase C — plan-modifier hygiene"
	$(GO) test -count=1 -run '^TestRequiresReplaceRespectsUseStateForUnknown$$' ./internal/provider/
	$(GO) test -count=1 -run '^TestOptionalComputedHasUseStateForUnknown$$' ./internal/provider/
	$(GO) test -count=1 -run '^TestPEMEquivalent' ./internal/planmodifiers/
	@echo "==> Phase C — request ID correlation"
	$(GO) test -count=1 -run '^(TestNewRequestID|TestDoRequest_EmitsXRequestIDHeader|TestDoRequest_RetriesShareRequestID)' ./internal/client/
	@echo "==> Phase D — destroy protection safety rail"
	$(GO) test -count=1 -run '^TestDestroyProtection' ./internal/client/
	$(GO) test -count=1 -run '^TestProvider_Configure_(DestroyProtection|SafeApply)' ./internal/provider/
	@echo "==> Phase E — config-time cross-attribute validators"
	$(GO) test -count=1 -run '^TestRequiredWhenEqual' ./internal/resourcevalidators/
	$(GO) test -count=1 -run '^TestConfigValidatorsCoverage$$' ./internal/provider/
	@echo "==> Phase F — plan-time destroy warnings"
	$(GO) test -count=1 -run '^TestWarnOnDestroy' ./internal/planhelpers/
	$(GO) test -count=1 -run '^TestDestroyWarningCoverage$$' ./internal/provider/
	@echo "==> Phase G — secret redaction in error diagnostics"
	$(GO) test -count=1 -run '^(TestIsSensitiveKey|TestRedact|TestAPIErrorBodyNeverLeaksSecrets|TestDoOnceRedacts)' ./internal/client/
	@echo "==> Phase H — strict static analysis (golangci-lint, 18 linters)"
	@test -x "$(GOLANGCI_LINT)" || { \
		echo "ERROR: golangci-lint not found at $(GOLANGCI_LINT)"; \
		echo "Install via:"; \
		echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"; \
		exit 1; \
	}
	$(GOLANGCI_LINT) run --timeout=5m $(PKGS)
	@echo "==> Phase I — docs & examples coverage ratchet"
	$(GO) test -count=1 -run '^TestDocs(Coverage|NoPlaceholders)$$' ./internal/provider/
	@echo "==> Phase J — acceptance test coverage ratchet"
	$(GO) test -count=1 -run '^TestAcceptanceTestCoverage$$' ./internal/provider/
	@echo
	@echo "All Phase B+C+D+E+F+G+H+I+J battle-hardening invariants green — safe to tag."

## testacc: Run acceptance tests against a real TrueNAS instance. Requires TRUENAS_URL, TRUENAS_API_KEY.
testacc:
	TF_ACC=1 $(GO) test -v -count=1 -timeout 120m $(INTERNAL)

## fmt: Format all Go files with gofmt.
fmt:
	$(GOFMT) -w -s .

## fmtcheck: Fail if any Go files are not gofmt-clean.
fmtcheck:
	@UNFORMATTED=$$($(GOFMT) -l . | grep -v '^\.go/' || true); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "Files not formatted:"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi

## vet: Run go vet on all packages.
vet:
	$(GO) vet $(PKGS)

## lint: Run golangci-lint across all packages.
lint:
	$(GOLANGCI_LINT) run --timeout=5m $(PKGS)

## tidy: Run go mod tidy.
tidy:
	$(GO) mod tidy

## docs: Validate Terraform Registry documentation layout. Non-destructive.
## Prefer this over `docs-regen`; the hand-authored docs carry custom
## subcategory and prose that `tfplugindocs generate` strips.
docs:
	$(TFPLUGINDOCS) validate --provider-name truenas ./

## docs-regen: DANGEROUS — regenerate docs from scratch. Strips custom
## subcategory/prose; only use when bulk-bootstrapping a new resource
## or after a schema-wide attribute rename. Review the diff carefully
## before committing; most of the time you want `make docs` only.
docs-regen:
	$(TFPLUGINDOCS) generate --provider-name truenas

## clean: Remove build artifacts.
clean:
	rm -f $(BINARY) coverage.out
	rm -rf dist/

## help: Print this help.
help:
	@echo "Usage: make <target>"
	@echo
	@echo "Targets:"
	@grep -E '^##' $(MAKEFILE_LIST) | sed -e 's/## /  /'
