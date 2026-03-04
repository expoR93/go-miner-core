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

// Work describes a unit of mining work derived from a live block header.
// Clients will populate this from whatever API or pool they are using.
//
// Only the fields required to assemble and hash the header are included; the
// nonce range can either be derived externally or managed by the engine itself.
type Work struct {
	Version    uint32   // block version
	PrevHash   [32]byte // little‑endian previous block hash
	MerkleRoot [32]byte // little‑endian merkle root
	Timestamp  uint32   // header time; miners may adjust +‑2h
	Bits       uint32   // compact target representation from the network
	StartNonce uint64   // optional starting nonce (usually 0)
}

// Config holds the parameters for the mining engine.
// Workers specifies how many goroutines will share the nonce space; the
// Work field carries the actual header information the engine will use to
// build candidate headers.
//
// Previously we stored a difficulty integer; we now rely on the network's
// compact “bits“ value and expose helper methods to translate it to a
// big.Int target or human‑readable difficulty as needed.
type Config struct {
	Workers int
	Work    Work
}
