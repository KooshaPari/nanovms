module github.com/kooshapari/nanovms

go 1.23.0

toolchain go1.23.4

require (
	go.uber.org/mock v0.6.0
	gopkg.in/yaml.v3 v3.0.1
)

replace go.uber.org/mock => ./third_party/go.uber.org/mock
