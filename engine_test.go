package gominer

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"sync/atomic"
	"testing"
	"time"
)

// helper that independently performs a double SHA256 so we can verify the
// library implementation is correct without just calling the function under
// test.
func referenceDoubleSHA256(nonce uint64) [32]byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, nonce)
	first := sha256.Sum256(b)
	return sha256.Sum256(first[:])
}

func TestPerformDoubleSHA256(t *testing.T) {
	for _, n := range []uint64{0, 1, 0xdeadbeef, 0xffffffffffffffff} {
		got := performDoubleSHA256(n)
		want := referenceDoubleSHA256(n)
		if got != want {
			t.Fatalf("nonce %d: expected %x, got %x", n, want, got)
		}
	}
}

func TestCompactToBig(t *testing.T) {
	cases := []struct {
		compact  uint32
		expected string // decimal string of the big int
	}{
		// exponent <=3 (shift right)
		{compact: 0x02080000, expected: "2048"},
		{compact: 0x03000001, expected: "1"},
		{compact: 0x030000ff, expected: "255"},
		// exponent >3 (shift left)
		{compact: 0x04000001, expected: "256"},   // 1 << 8
		{compact: 0x05000001, expected: "65536"}, // 1 << 16
		{compact: 0x06012345, expected: new(big.Int).Lsh(new(big.Int).SetUint64(0x12345), 8*(6-3)).String()},
		// very large exponent (should still return a big.Int larger than any 256-bit)
		{compact: 0xff000001, expected: new(big.Int).Lsh(new(big.Int).SetUint64(1), 8*(0xff-3)).String()},
	}

	for _, c := range cases {
		got := compactToBig(c.compact)
		if got.String() != c.expected {
			t.Errorf("compact 0x%08x: expected %s, got %s", c.compact, c.expected, got)
		}
	}
}

func TestIsBelowTarget(t *testing.T) {
	e := &Engine{}

	// create a target that is exactly the value of some "hash" so we can test
	// both sides of the comparison.  we'll just use a big.Int supplied by us.
	target := big.NewInt(0)
	target.SetUint64(0x123456)

	// hash equal to target should *not* be considered below
	var equalHash [32]byte
	equalHash[31] = 0x12
	equalHash[30] = 0x34
	equalHash[29] = 0x56

	if e.isBelowTarget(equalHash, target) {
		t.Errorf("hash equal to target reported below target")
	}

	// hash smaller
	var small [32]byte
	small[31] = 0x01
	if !e.isBelowTarget(small, target) {
		t.Errorf("hash that is smaller than target not reported below")
	}

	// hash larger: make a value with a leading byte == 0xff
	var large [32]byte
	large[0] = 0xff
	if e.isBelowTarget(large, target) {
		t.Errorf("large hash incorrectly considered below target")
	}
}

func TestStatsSnapshot(t *testing.T) {
	e := &Engine{}
	e.start = time.Now().Add(-time.Second) // pretend we started a second ago
	atomic.StoreUint64(&e.stats.TotalHashes, 100)
	atomic.StoreUint64(&e.stats.BlocksFound, 2)

	s := e.Stats()
	if s.TotalHashes != 100 {
		t.Fatalf("expected 100 hashes, got %d", s.TotalHashes)
	}
	if s.BlocksFound != 2 {
		t.Fatalf("expected 2 blocks found, got %d", s.BlocksFound)
	}
	if s.Uptime < time.Second {
		t.Fatalf("uptime too low: %s", s.Uptime)
	}
	if s.HashRate <= 0 {
		t.Fatalf("hash rate should be positive")
	}
}

func TestEngineStartFindsBlock(t *testing.T) {
	// pick a bits value that yields an enormous target (much larger than any
	// 256-bit hash).  using exponent=0xff guarantees the left shift goes well
	// beyond 256 bits.
	cfg := Config{
		Workers: 2,
		Work:    Work{Bits: 0xff000001},
	}
	e := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	found := make(chan BlockHeader, 1)
	go e.Start(ctx, found)

	select {
	case bh := <-found:
		if bh.Nonce != 0 {
			t.Logf("unexpected nonce %d, but we don't really care", bh.Nonce)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected a header but none was produced")
	}

	stats := e.Stats()
	if stats.TotalHashes == 0 {
		t.Errorf("expected some hashes, got zero")
	}
	if stats.BlocksFound == 0 {
		t.Errorf("expected at least one block found")
	}
}

func TestStartCancellation(t *testing.T) {
	cfg := Config{Workers: 4, Work: Work{Bits: 0xff000001}}
	e := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	found := make(chan BlockHeader, 1)
	e.Start(ctx, found)

	select {
	case bh := <-found:
		t.Fatalf("should not have produced a header, got %+v", bh)
	case <-time.After(100 * time.Millisecond):
		// okay
	}
}

func TestMinerStopsOnContext(t *testing.T) {
	// direct exercise of miner to hit the <ctx.Done> branch
	e := &Engine{}
	target := big.NewInt(0)
	target.SetUint64(0) // impossible to beat

	ctx, cancel := context.WithCancel(context.Background())
	found := make(chan BlockHeader, 1)

	go func() {
		// cancel after a short delay so miner stops mid-loop
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	// only search a small nonce range so test completes quickly
	e.miner(ctx, 0, 0, 100000, target, found)

	// after the miner returns, we expect there to be no block and stats may have
	// some hashes counted but not necessarily equal to range size (context
	// might have stopped early).
	if atomic.LoadUint64(&e.stats.BlocksFound) != 0 {
		t.Fatalf("no blocks should have been found")
	}
}

// Benchmarks

func BenchmarkPerformDoubleSHA256(b *testing.B) {
	// cycle through nonces to avoid compiler optimizations
	for i := 0; i < b.N; i++ {
		_ = performDoubleSHA256(uint64(i))
	}
}

func BenchmarkCompactToBig(b *testing.B) {
	// use a fixed bit pattern and vary the loop index slightly
	var c uint32 = 0x1d00ffff
	for i := 0; i < b.N; i++ {
		_ = compactToBig(c + uint32(i))
	}
}

func BenchmarkIsBelowTarget(b *testing.B) {
	e := &Engine{}
	target := big.NewInt(0)
	target.SetUint64(0xabcdef)
	var hash [32]byte
	for i := 0; i < b.N; i++ {
		// modify hash so that it sometimes lies below the target
		hash[31] = byte(i)
		_ = e.isBelowTarget(hash, target)
	}
}

func BenchmarkStats(b *testing.B) {
	e := &Engine{}
	e.start = time.Now()
	for i := 0; i < b.N; i++ {
		_ = e.Stats()
	}
}

func BenchmarkEngineMiner(b *testing.B) {
	e := &Engine{}
	ctx := context.Background()
	target := big.NewInt(0) // unreachable target
	results := make(chan BlockHeader, 1)
	const rangeSize = 1000000
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.miner(ctx, 0, 0, rangeSize, target, results)
	}
}
