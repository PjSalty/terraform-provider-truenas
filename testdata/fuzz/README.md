# Fuzz corpus

Go native fuzzing persists regression inputs here automatically. Whenever a
CI fuzz run finds a crashing input, it is committed as a file under
`testdata/fuzz/<FuzzName>/<hash>` so that subsequent regular `go test` runs
replay it as a regression test.

## Running fuzz tests

Short smoke run (CI default, 30s per target):

```bash
go test -run='^$' -fuzz='^FuzzParseRetryAfter$' -fuzztime=30s ./internal/client/
```

Long campaign (local, until you ctrl-c):

```bash
go test -run='^$' -fuzz='^FuzzNormalizeJSON$' -fuzztime=1h ./internal/resources/
```

## Current fuzz targets

| Package | Target | What it checks |
|---------|--------|----------------|
| internal/client | FuzzParseRetryAfter | HTTP Retry-After header parsing never panics |
| internal/validators | FuzzIPOrCIDR | IPv4/v6/CIDR validator never panics |
| internal/validators | FuzzHostOrIP | Host-or-IP validator never panics |
| internal/validators | FuzzZFSPath | `/mnt/...` path validator never panics |
| internal/validators | FuzzCompressionAlgorithm | Dataset compression enum validator never panics |
| internal/resources | FuzzNormalizeJSON | JSON canonicalization never panics |
| internal/resources | FuzzStripJSONNulls | Null-value stripping never panics |
| internal/resources | FuzzFilterJSONByKeys | Key-filtered JSON projection never panics |

## Property under test

All targets assert a single property: **the function never panics on any
input**. Output correctness is verified by regular unit tests — fuzz tests
exist only to find panics, out-of-bounds accesses, infinite loops, and
stack exhaustion.
