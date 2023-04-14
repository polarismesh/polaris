#### Discover Services

```
// goos: linux
// goarch: amd64
// pkg: github.com/polarismesh/polaris/test/benchmark/grpc
// cpu: AMD EPYC 7K62 48-Core Processor


// Benchmark_DiscoverServicesWithoutRevision-16
//      763           1599653 ns/op
// --- BENCH: Benchmark_DiscoverServicesWithoutRevision-16
//     discover_test.go:100: connection server success
//     discover_test.go:106: create discover client success
//     discover_test.go:100: connection server success
//     discover_test.go:106: create discover client success
//     discover_test.go:100: connection server success
//     discover_test.go:106: create discover client success

// Benchmark_DiscoverServicesWithRevision-16
//     4984            235381 ns/op
// --- BENCH: Benchmark_DiscoverServicesWithRevision-16
//     discover_test.go:100: connection server success
//     discover_test.go:106: create discover client success
//     discover_test.go:100: connection server success
//     discover_test.go:106: create discover client success
//     discover_test.go:100: connection server success
//     discover_test.go:106: create discover client success
// PASS
// ok      github.com/polarismesh/polaris/test/benchmark/grpc      2.905s
```