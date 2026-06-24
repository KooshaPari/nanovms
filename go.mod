module github.com/kooshapari/nanovms

go 1.23.0

toolchain go1.23.4

require (
	go.uber.org/mock v0.6.0
	gopkg.in/yaml.v3 v3.0.1
)

// `go.uber.org/mock` is used by `internal/adapters/linux` for syscall
// mocks. We resolve it from the in-tree vendored copy under
// `third_party/go.uber.org/mock` to keep builds reproducible without
// network access and to pin the exact version. The vendored copy is a
// clean mirror of upstream `go.uber.org/mock v0.6.0`.
replace go.uber.org/mock => ./third_party/go.uber.org/mock
