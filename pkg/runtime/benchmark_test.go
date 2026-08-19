package runtime

import "testing"

func BenchmarkRegistryGet(b *testing.B) {
	r := NewRegistry()
	r.Register(NewMemoryBackend())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Get("memory")
	}
}

func BenchmarkRegistryAll(b *testing.B) {
	r := NewRegistry()
	r.Register(NewMemoryBackend())
	r.Register(NewFirecrackerBackend())
	r.Register(NewGvisorBackend())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.All()
	}
}

func BenchmarkBackendRegistryResolve(b *testing.B) {
	br := NewBackendRegistry()
	br.Register("memory", NewMemoryBackend())
	br.Register("firecracker", NewFirecrackerBackend())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Resolve("memory")
	}
}

func BenchmarkParseBackend(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseBackend("memory")
	}
}

func BenchmarkSupports(b *testing.B) {
	backend := NewMemoryBackend()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backend.Supports(FeatureIsolation)
	}
}
