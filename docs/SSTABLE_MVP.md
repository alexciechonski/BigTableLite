# SSTable Engine MVP

This document describes the C++ SSTable storage engine implementation for BigTableLite.

## Architecture

The SSTable engine consists of:

1. **Memtable**: In-memory sorted map (`std::map<std::string, std::string>`) that stores recent writes
2. **SSTable Writer**: Flushes memtable to disk in sorted order with an index
3. **SSTable Reader**: Reads from disk using binary search on the index
4. **C API**: C wrappers exposed to Go via cgo

## File Structure

```
sstable/
  ├── sstable.cpp    # C++ implementation
  ├── sstable.h      # C API header
  └── Makefile       # Build static library

pkg/storage/
  ├── sstable.go     # Go cgo bindings
  └── sstable_test.go # Tests

data/                # SSTable files directory (created at runtime)
  ├── sstable_0001.sst
  ├── sstable_0002.sst
  └── ...
```

## SSTable File Format

Each SSTable file contains:

1. **Data Section**: `<key_len><key><value_len><value>...` (sorted by key)
2. **Index Section**: `<num_entries><key><offset>...` (sorted by key)
3. **Index Start Offset**: 8-byte offset at end of file pointing to index start

## Building

```bash
# Build C++ library and Go binary
make build

# Or manually:
cd sstable && make
cd .. && CGO_ENABLED=1 go build -o bigtablelite ./cmd/server
```

## Usage

### Default (SSTable backend):
```bash
./bigtablelite -data-dir ./data
```

### Redis backend (for comparison):
```bash
./bigtablelite -use-redis -redis-addr localhost:6379
```

## Features

Memtable with automatic flushing at 1MB threshold  
SSTable file format with index for efficient lookups  
Binary search on index for O(log n) lookups  
Reads from memtable first, then SSTables (newest to oldest)  
Persistent storage on disk  
cgo integration with Go  

## Limitations (MVP)

The MVP does NOT include:
- Write-Ahead Log (WAL)
- Bloom filters
- Block indexes
- Compression
- Compaction (major/minor)
- Multi-threaded writes
- On-disk caching layers
- File metadata/versioning

## Testing

```bash
# Run SSTable tests
go test ./pkg/storage -v

# Run all tests
make test
```

## Performance Notes

- Memtable operations: O(log n) for insert/lookup
- SSTable lookups: O(log n) binary search on index
- Flush operations: O(n) sequential write
- No compaction means SSTable count grows over time (future enhancement)

