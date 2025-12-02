#include "sstable.h"
#include <map>
#include <string>
#include <fstream>
#include <iostream>
#include <algorithm>
#include <cstring>
#include <vector>
#include <cstdint>

// Memtable implementation using std::map
static std::map<std::string, std::string> memtable;
static size_t memtable_size = 0;
static const size_t MEMTABLE_FLUSH_THRESHOLD = 1024 * 1024; // 1 MB
static uint32_t sstable_counter = 0;
static std::string data_dir = "./data";

// Helper to calculate size of a key-value pair
static size_t calculate_kv_size(const std::string& key, const std::string& value) {
    return key.size() + sizeof(uint32_t) + value.size();
}

// Initialize SSTable engine
extern "C" bool sstable_init(const char* dir) {
    if (dir != nullptr) {
        data_dir = std::string(dir);
    }
    
    // Ensure data directory exists
    std::string mkdir_cmd = "mkdir -p " + data_dir;
    system(mkdir_cmd.c_str());
    
    // Find the highest existing SSTable number
    sstable_counter = 0;
    for (uint32_t i = 1; i < 10000; i++) {
        char filename[256];
        snprintf(filename, sizeof(filename), "%s/sstable_%04u.sst", data_dir.c_str(), i);
        std::ifstream file(filename);
        if (file.good()) {
            sstable_counter = i;
        } else {
            break;
        }
    }

    memtable.clear();
    memtable_size = 0;
    
    return true;
}

// sstable destructor
extern "C" void sstable_destroy() {
    memtable.clear();
    memtable_size = 0;
    sstable_counter = 0;
}

// Put a key-value pair into memtable
extern "C" bool sstable_put(const char* key, const char* value) {
    if (key == nullptr || value == nullptr) {
        return false;
    }
    
    std::string key_str(key);
    std::string value_str(value);
    
    // Calculate size change
    size_t old_size = 0;
    if (memtable.find(key_str) != memtable.end()) {
        old_size = calculate_kv_size(key_str, memtable[key_str]);
    }
    size_t new_size = calculate_kv_size(key_str, value_str);
    
    memtable_size = memtable_size - old_size + new_size;
    memtable[key_str] = value_str;
    
    return true;
}

// Get a value from memtable
extern "C" bool sstable_get_memtable(const char* key, sstable_bytes* out) {
    if (key == nullptr || out == nullptr) {
        return false;
    }
    
    std::string key_str(key);
    auto it = memtable.find(key_str);
    if (it != memtable.end()) {
        // Allocate memory for the value
        size_t len = it->second.size();
        char* data = new char[len];
        std::memcpy(data, it->second.c_str(), len);
        
        out->data = data;
        out->len = len;
        return true;
    }
    
    return false;
}

// Check if memtable needs flushing
extern "C" bool sstable_needs_flush() {
    return memtable_size >= MEMTABLE_FLUSH_THRESHOLD;
}

// Write memtable to SSTable file
extern "C" bool sstable_flush() {
    if (memtable.empty()) {
        return true; // Nothing to flush
    }
    
    // Generate filename
    sstable_counter++;
    char filename[256];
    snprintf(filename, sizeof(filename), "%s/sstable_%04u.sst", data_dir.c_str(), sstable_counter);
    
    std::ofstream file(filename, std::ios::binary);
    if (!file.is_open()) {
        return false;
    }
    
    // Write data section: <key><value_length><value>...
    std::vector<std::pair<std::string, size_t>> index; // key -> offset
    size_t current_offset = 0;
    
    for (const auto& kv : memtable) {
        // Record offset for index
        index.push_back({kv.first, current_offset});
        
        // Write key
        uint32_t key_len = kv.first.size();
        file.write(reinterpret_cast<const char*>(&key_len), sizeof(key_len));
        file.write(kv.first.c_str(), key_len);
        
        // Write value length and value
        uint32_t value_len = kv.second.size();
        file.write(reinterpret_cast<const char*>(&value_len), sizeof(value_len));
        file.write(kv.second.c_str(), value_len);
        
        current_offset += sizeof(key_len) + key_len + sizeof(value_len) + value_len;
    }
    
    // Write index section: <num_entries><key><offset>...
    size_t index_start = current_offset;
    uint32_t num_entries = index.size();
    file.write(reinterpret_cast<const char*>(&num_entries), sizeof(num_entries));
    
    for (const auto& entry : index) {
        uint32_t key_len = entry.first.size();
        file.write(reinterpret_cast<const char*>(&key_len), sizeof(key_len));
        file.write(entry.first.c_str(), key_len);
        
        size_t offset = entry.second;
        file.write(reinterpret_cast<const char*>(&offset), sizeof(offset));
    }
    
    // Write index start offset at the end
    file.write(reinterpret_cast<const char*>(&index_start), sizeof(index_start));
    
    file.close();
    
    // Clear memtable
    memtable.clear();
    memtable_size = 0;
    
    return true;
}

// Read from a single SSTable file
static bool read_sstable(const char* filename, const char* key, std::string& out_value) {
    std::ifstream file(filename, std::ios::binary);
    if (!file.is_open()) {
        return false;
    }
    
    // Read index start offset from end of file
    file.seekg(-static_cast<int>(sizeof(size_t)), std::ios::end);
    size_t index_start;
    file.read(reinterpret_cast<char*>(&index_start), sizeof(index_start));
    
    // Read index into memory
    file.seekg(index_start, std::ios::beg);
    uint32_t num_entries;
    file.read(reinterpret_cast<char*>(&num_entries), sizeof(num_entries));
    
    // Load entire index into memory
    std::vector<std::pair<std::string, size_t>> index;
    for (uint32_t i = 0; i < num_entries; i++) {
        uint32_t klen;
        file.read(reinterpret_cast<char*>(&klen), sizeof(klen));
        std::string index_key(klen, '\0');
        file.read(&index_key[0], klen);
        size_t offset;
        file.read(reinterpret_cast<char*>(&offset), sizeof(offset));
        index.push_back({index_key, offset});
    }
    
    // Binary search in loaded index
    std::string key_str(key);
    int left = 0, right = index.size() - 1;
    size_t target_offset = SIZE_MAX;
    
    while (left <= right) {
        int mid = (left + right) / 2;
        int cmp = key_str.compare(index[mid].first);
        if (cmp == 0) {
            target_offset = index[mid].second;
            break;
        } else if (cmp < 0) {
            right = mid - 1;
        } else {
            left = mid + 1;
        }
    }
    
    if (target_offset == SIZE_MAX) {
        return false; // Key not found
    }
    
    // Read value at target_offset
    file.seekg(target_offset, std::ios::beg);
    
    // Read key (skip it)
    uint32_t key_len;
    file.read(reinterpret_cast<char*>(&key_len), sizeof(key_len));
    file.seekg(key_len, std::ios::cur);
    
    // Read value
    uint32_t value_len;
    file.read(reinterpret_cast<char*>(&value_len), sizeof(value_len));
    out_value.resize(value_len);
    file.read(&out_value[0], value_len);
    
    return true;
}

// Get value from SSTables (newest to oldest)
extern "C" bool sstable_get(const char* key, sstable_bytes* out) {
    if (key == nullptr || out == nullptr) {
        return false;
    }
    
    // First check memtable
    if (sstable_get_memtable(key, out)) {
        return true;
    }
    
    // Then check SSTables from newest to oldest
    std::string value;
    for (uint32_t i = sstable_counter; i >= 1; i--) {
        char filename[256];
        snprintf(filename, sizeof(filename), "%s/sstable_%04u.sst", data_dir.c_str(), i);
        
        if (read_sstable(filename, key, value)) {
            // Allocate memory for the value
            size_t len = value.size();
            char* data = new char[len];
            std::memcpy(data, value.c_str(), len);
            
            out->data = data;
            out->len = len;
            return true;
        }
    }
    
    return false;
}

// Free memory allocated by sstable_get
extern "C" void sstable_free_bytes(sstable_bytes* bytes) {
    if (bytes != nullptr && bytes->data != nullptr) {
        delete[] bytes->data;
        bytes->data = nullptr;
        bytes->len = 0;
    }
}

