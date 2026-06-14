module github.com/kooshapari/nanovms

go 1.23.0

toolchain go1.23.4

require (
	github.com/kooshapari/pheno-go-ctxkit v0.0.0
	go.uber.org/mock v0.6.0
	gopkg.in/yaml.v3 v3.0.1
)

replace (
	github.com/kooshapari/pheno-go-ctxkit => ../.worktrees/l3-52-pheno-go-ctxkit-2026-06-11/pheno-go-ctxkit
	go.uber.org/mock => ./third_party/go.uber.org/mock
)
