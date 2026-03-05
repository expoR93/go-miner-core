# 🚀 go-miner-core
A Bitcoin Mining Engine.

**A High-Performance, Standalone Bitcoin PoW Hashing Engine built in Golang.**

Go-Miner-Core is a decoupled mining library designed for real-time Bitcoin simulation.
It leverages Go's low-level concurrency primitives to provide a thread-safe, high-throughput
hashing pipeline.

**Note:** when `Start` is invoked the engine will automatically cancel remaining workers
as soon as the first valid block header is found; callers only need to cancel the
context for external events such as new work or shutdown.

## 🛠️ Key Features
* **Parallel Hashing Engine:** Uses a **Static Range Partitioning** strategy to distribute the
64-bit nonce search space across multiple CPU cores.
* **Double-SHA256 Pipeline:** Optimized implementation of the Bitcoin PoW algorithm.
* **Telemetry via atomics:** Uses `sync/atomic` to increment counters safely; a `Stats()` method
returns a snapshot (`WorkerStats`) that can be polled by a UI layer.  The implementation
is straightforward and easy to explain.
* **Graceful Orchestration:** Full support for `context.Context` for safe, leak-free shutdowns.

## 🏗️ System Architecture
The engine is designed as a "Headless" library. It manages its own goroutine lifecycle and 
communicates via:
1. **Channels:** For event-driven "Block Found" notifications.
2. **Snapshots:** For non-blocking telemetry retrieval (ideal for TUIs or Dashboards).

## 🚀 Quick start
```go
import "github.com/expoR93/go-miner-core"

func main() {
    work := gominer.Work{
        Version:    0x20000000, // example
        PrevHash:   [32]byte{/* ... */},
        MerkleRoot: [32]byte{/* ... */},
        Timestamp:  uint32(time.Now().Unix()),
        Bits:       0x1d00ffff, // difficulty 1
        StartNonce: 0,
    }

    engine := gominer.New(gominer.Config{
        Workers: 8,
        Work:    work,
    })

    go engine.Start(ctx)
}
```

### 📈 UI telemetry example
A UI layer (TUI, web dashboard, etc.) can poll engine statistics without
blocking miners:

```go
// somewhere in your render loop or HTTP handler
stats := engine.Stats() // quick snapshot, thread-safe
fmt.Printf("hashrate: %.2f h/s, uptime: %s\n", stats.HashRate, stats.Uptime)
```

The returned `WorkerStats` struct contains total hashes, blocks found, uptime,
and a calculated hash rate.

## 📊 Performance 
Run the benchmarks to see throughput and memory usage on your machine:

```bash
# run all benchmarks with memory statistics
go test -run=^$ -bench=. -benchmem
```

A typical result on a modern laptop looks like:

```
goos: linux
goarch: amd64
pkg: github.com/expoR93/go-miner-core
cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz
BenchmarkPerformDoubleSHA256-3           1796482               675.8 ns/op      0 B/op           0 allocs/op
BenchmarkCompactToBig-3                  5931331               234.5 ns/op    120 B/op           3 allocs/op
BenchmarkIsBelowTarget-3                12348746               106.3 ns/op     64 B/op           1 allocs/op
BenchmarkStats-3                        35038653                39.09 ns/op      0 B/op           0 allocs/op
BenchmarkEngineMiner-3                         1        1110451578 ns/op       64000384 B/op     1000004 allocs/op
```

The library provides lightweight, focused benchmarks for the core hashing
pipeline (`performDoubleSHA256`), target conversion, comparison logic, and a
shared `miner` workload; you can extend them or add new ones as the engine
evolves.

Built for the Go-Miner Simulation Project.

