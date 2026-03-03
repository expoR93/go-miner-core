package gominer

import "time"

// BlockHeader represents the result of a successful mining attempt.
// Exported fields allow the Client to render these details.
type BlockHeader struct {
	Timestamp time.Time
	Nonce     uint64
	Hash      string
	WorkerID  int
}

// WorkerStats provides a snapshot of the engine's current performance.
// We use these types to pass data across the "De-coupled" boundary.
type WorkerStats struct {
	TotalHashes uint64
	BlocksFound uint64
	Uptime      time.Duration
	HashRate    float64 // Hahses per second
}

// Config holds the parameters for the mining engine
type Config struct {
	Workers    int
	Difficulty int
}
