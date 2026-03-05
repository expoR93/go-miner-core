package gominer

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"
)

type Engine struct {
	config Config
	stats  WorkerStats // We'll update fields here atomically
	start  time.Time   // when mining began; used to compute uptime
}

func New(cfg Config) *Engine {
	return &Engine{config: cfg}
}

// Start spawns workers and manages the mining lifecycle.  It listens for the
// first header produced by any worker and cancels the internal context so the
// remaining goroutines exit promptly.  The caller may still cancel the parent
// context (e.g. when receiving new work) to interrupt mining for other reasons.
func (e *Engine) Start(ctx context.Context, found chan<- BlockHeader) {
	var wg sync.WaitGroup

	// record when mining started so we can compute uptime later
	e.start = time.Now()

	miningCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// internal channel used to observe the very first result so we can cancel
	// the remaining workers without relying on the caller to do so.  We buffer
	// it so that a fast worker doesn't block trying to send while the watcher
	// goroutine is still starting.
	results := make(chan BlockHeader, 1)

	go func() {
		select {
		case <-miningCtx.Done():
			// nothing to do, shutdown already in progress
		case bh := <-results:
			// forward the header to the external channel and cancel the context.
			// if the caller isn't ready to receive, we don't block miners; the
			// buffered send above prevents this race.
			select {
			case found <- bh:
			default:
			}
			cancel()
		}
	}()

	// compute the target once from the compact bits field supplied with the work
	target := compactToBig(e.config.Work.Bits)

	// Splits the uint64 space (0 to 18,446,744,073,709,551,615)
	const maxNonce uint64 = ^uint64(0)

	rangeSize := maxNonce / uint64(e.config.Workers)

	for i := 0; i < e.config.Workers; i++ {
		wg.Add(1)
		// incorporate any configured starting nonce (e.g. from a pool job)
		start := e.config.Work.StartNonce + uint64(i)*rangeSize
		end := start + rangeSize

		go func(id int, s, hide uint64) {
			defer wg.Done()
			e.miner(miningCtx, id, s, hide, target, results)
		}(i, start, end)
	}

	wg.Wait()
}

func (e *Engine) miner(ctx context.Context, id int, start, end uint64, target *big.Int, results chan<- BlockHeader) {
	for nonce := start; nonce < end; nonce++ {
		select {
		case <-ctx.Done():
			return
		default:
			// Double SHA256
			hash := performDoubleSHA256(nonce)

			// Increment global hash count atomically
			atomic.AddUint64(&e.stats.TotalHashes, 1)

			// compare hash to precomputed target
			if e.isBelowTarget(hash, target) {
				atomic.AddUint64(&e.stats.BlocksFound, 1)
				results <- BlockHeader{
					Timestamp: time.Now(),
					Nonce:     nonce,
					Hash:      fmt.Sprintf("%x", hash),
					WorkerID:  id,
				}
				return
			}
		}
	}
}

func performDoubleSHA256(nonce uint64) [32]byte {
	b := make([]byte, 8)

	binary.LittleEndian.PutUint64(b, nonce)

	first := sha256.Sum256(b)

	return sha256.Sum256(first[:])
}

// compactToBig decodes Bitcoin's "compact" target format (bits) into a
// full 256-bit target.  This mirrors the logic used by bitcoind's
// `arith_uint256::SetCompact` and is what mining pools deliver in the
// block header.
func compactToBig(compact uint32) *big.Int {
	exponent := uint(compact >> 24)
	mantissa := compact & 0x007fffff

	result := new(big.Int).SetUint64(uint64(mantissa))
	if exponent <= 3 {
		result.Rsh(result, 8*(3-exponent))
	} else {
		result.Lsh(result, 8*(exponent-3))
	}
	return result
}

// Stats returns an up‑to‑date snapshot of the engine's performance.  It's
// safe for concurrent callers and is intended for UI/monitoring layers that
// poll periodically.
func (e *Engine) Stats() WorkerStats {
	total := atomic.LoadUint64(&e.stats.TotalHashes)
	blocks := atomic.LoadUint64(&e.stats.BlocksFound)
	uptime := time.Since(e.start)
	hashRate := 0.0
	if uptime.Seconds() > 0 {
		hashRate = float64(total) / uptime.Seconds()
	}
	return WorkerStats{
		TotalHashes: total,
		BlocksFound: blocks,
		Uptime:      uptime,
		HashRate:    hashRate,
	}
}

// isBelowTarget converts the [32]byte hash to a big.Int and compares it to our target
func (e *Engine) isBelowTarget(hash [32]byte, target *big.Int) bool {
	var hashInt big.Int

	// Bitcoin hashes are read as little-endian, but for math/big comparison,
	// we usually treat the 32-byte slice as a big-endian unsigned integer
	hashInt.SetBytes(hash[:])

	// If hashInt < target, we found the block!
	return hashInt.Cmp(target) == -1
}
