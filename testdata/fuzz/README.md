# Fuzz corpus

Go native fuzzing persists regression inputs here automatically. Whenever a
CI fuzz run finds a crashing input, it is committed as a file under
`testdata/fuzz/<FuzzName>/<hash>` so that subsequent regular `go test` runs
replay it as a regression test.

## Running fuzz tests

List every target:

```bash
grep -rho 'func Fuzz[A-Za-z0-9_]*' internal/*/[a-z]*_test.go | sed 's/func //' | sort
```

Short smoke run (CI default, 30s per target):

```bash
go test -run='^$' -fuzz='^FuzzIsNotFound$' -fuzztime=30s ./internal/wsclient/
```

Long campaign (local, until you ctrl-c):

```bash
go test -run='^$' -fuzz='^FuzzNormalizeJSON$' -fuzztime=1h ./internal/resources/
```

Only one target can run at a time; `-fuzz` takes a single regex and Go refuses
a pattern matching more than one.

## Current fuzz targets

Counted rather than listed by name, because the per-type targets are generated
one per API struct and a hand-written list of them goes stale immediately. The
counts are asserted by `TestFuzzReadmeMatchesTargets`.

| Package | Targets | What they check |
|---------|--------:|-----------------|
| internal/types | 86 | one per API struct: unmarshalling any input never panics |
| internal/wsclient | 5 | JSON-RPC envelope, job update, and error classification parsing |
| internal/validators | 4 | IP/CIDR, host-or-IP, ZFS path, and compression enum validators |
| internal/resources | 3 | JSON canonicalization, null stripping, key-filtered projection |
| internal/customtypes | 1 | semantic YAML equality |

## Property under test

All targets assert a single property: **the function never panics on any
input**. Output correctness is verified by regular unit tests, fuzz tests
exist only to find panics, out-of-bounds accesses, infinite loops, and
stack exhaustion.
