# 🚀 go-miner-core
A Bitcoin Mining Engine.

**A High-Performance, Standalone Bitcoin PoW Hashing Engine built in Golang.**

Go-Miner-Core is a decoupled mining library designed for real-time Bitcoin simulation.
It leverages Go's low-level concurrency primitives to provide a thread-safe, high-throughput
hashing pipeline.

## 🛠️ Key Features
* **Parallel Hashing Engine:** Uses a **Static Range Partitioning** strategy to distribute the
64-bit nonce search space across multiple CPU cores.
* **Double-SHA256 Pipeline:** Optimized implementation of the Bitcoin PoW algorithm.
* **Lock-Free Telemetry:** Utilizes `sync/atomic` for high-frequency stat updates, ensuring
zero contention between the engine and the UI layer.
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
    engine := gominer.New(gominer.Config{Workers: 8})
    go engine.Start(ctx)
}
```

## 📊 Performance 
Run the benchmarks to see the throughput on your machine:
go test -bench=. -benchmem

Built for the Go-Miner Simulation Project.
