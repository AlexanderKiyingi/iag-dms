package store

// seedMemory is intentionally a no-op.
//
// It used to populate the in-memory store with the DMS prototype dataset — six
// distributors, nine outlets and the orders, beats, claims and dispatches around them.
// Migration 0008_purge_demo_seed.sql removed every one of those rows from every
// environment, and config.seedOnEmpty now refuses to seed in production at all.
//
// STORE_MODE=memory therefore starts empty too, which keeps the two store backends
// behaving the same way. The function is kept so both call sites — newMemoryState and
// SeedPostgres — still have an obvious place for a legitimate seed to live.
func seedMemory(_ *memoryState) {}
