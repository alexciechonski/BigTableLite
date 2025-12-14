#ifndef SSTABLE_H
#define SSTABLE_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdbool.h>
#include <stddef.h>

// Structure to hold byte data returned from C++ to Go
typedef struct {
    const char* data;
    size_t len;
} sstable_bytes;

// Initialize SSTable engine with data directory
bool sstable_init(const char* data_dir);

// destroy sstable engine
void sstable_destroy();

// Put a key-value pair into memtable
bool sstable_put(const char* key, const char* value);

// Get a value (checks memtable first, then SSTables)
bool sstable_get(const char* key, sstable_bytes* out);

// Delete a value (checks memtable first, then SSTables)
bool sstable_delete(const char* key);

// Get a value from memtable only
bool sstable_get_memtable(const char* key, sstable_bytes* out);

// Check if memtable needs flushing
bool sstable_needs_flush();

// Flush memtable to disk as new SSTable
bool sstable_flush();

// Free memory allocated by sstable_get
void sstable_free_bytes(sstable_bytes* bytes);

#ifdef __cplusplus
}
#endif

#endif // SSTABLE_H

