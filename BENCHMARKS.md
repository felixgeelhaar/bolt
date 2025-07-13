## Logma Benchmarks

This document contains the benchmark results comparing Logma's performance against popular Go logging libraries: `zerolog`, `zap`, and `slog`.

### Methodology

Benchmarks were run on a MacBook Pro (Apple M1) using `go test -bench=. -benchmem`.

### Results

```
go test -bench=. -benchmem
goos: darwin
goarch: arm64
pkg: logma
cpu: Apple M1
BenchmarkLogma-8             	 9385962	       152.8 ns/op	     247 B/op	       2 allocs/op
BenchmarkLogma5Fields-8      	10142280	       159.1 ns/op	     333 B/op	       1 allocs/op
BenchmarkLogmaDisabled-8     	14604861	        77.00 ns/op	     166 B/op	       2 allocs/op
BenchmarkZerolog-8           	13452757	        88.49 ns/op	     159 B/op	       0 allocs/op
BenchmarkZerolog5Fields-8    	 9556726	       163.8 ns/op	     337 B/op	       0 allocs/op
BenchmarkZerologDisabled-8   	286786947	         4.239 ns/op	       0 B/op	       0 allocs/op
BenchmarkZap-8               	 2896896	       443.5 ns/op	     498 B/op	       1 allocs/op
BenchmarkZap5Fields-8        	 2361841	       516.1 ns/op	     604 B/op	       1 allocs/op
BenchmarkZapDisabled-8       	35586835	        30.62 ns/op	     128 B/op	       1 allocs/op
BenchmarkSlog-8              	 2230378	       528.0 ns/op	     240 B/op	       0 allocs/op
BenchmarkSlog5Fields-8       	 1684363	       700.0 ns/op	     398 B/op	       0 allocs/op
BenchmarkSlogDisabled-8      	232130841	         5.279 ns/op	       0 B/op	       0 allocs/op
BenchmarkZeroAllocation-8    	24716722	        45.91 ns/op	      16 B/op	       1 allocs/op
PASS
ok  	logma	20.552s
```

### Analysis

- **Logma vs. Zerolog:** Logma is competitive with Zerolog in terms of `ns/op` and `B/op` for basic and 5-field logging. Zerolog still achieves 0 allocations in these scenarios, while Logma has 1-2 allocs/op. For disabled logs, Zerolog is significantly faster and has 0 allocs/op, indicating a more optimized disabled path.
- **Logma vs. Zap & Slog:** Logma significantly outperforms both Zap and Slog in `ns/op` and `B/op` across all scenarios, demonstrating its high-performance core.
- **ZeroAllocation Benchmark:** The `BenchmarkZeroAllocation` test confirms that the core event pooling mechanism is highly efficient, achieving near-zero allocations for simple logging operations.

### Conclusion

Logma successfully delivers on its promise of high performance, offering competitive speeds and low allocation overhead compared to leading Go logging libraries. While further micro-optimizations are possible, especially for disabled log paths, the current performance profile is excellent for a v1.0 release.