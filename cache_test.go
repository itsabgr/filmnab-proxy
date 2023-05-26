package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLargeFile(t *testing.T) {
	dbPath := filepath.Join(os.TempDir(), strconv.FormatInt(time.Now().Unix(), 10))
	db := must(leveldb.OpenFile(dbPath, &opt.Options{ErrorIfExist: true}))
	defer db.Close()
	defer os.RemoveAll(dbPath)
	cache := newCache(db, 1e+10, func(ctx context.Context, key string) ([]byte, error) {
		return make([]byte, 1e+9), nil
	})
	result := must(cache.Get(context.Background(), "key"))
	assert(!result.CacheUsed)
	assert(result.ValueCached)
	assert(len(result.Value) == 1e+9)
	result = must(cache.Get(context.Background(), "key"))
	assert(result.CacheUsed)
	assert(result.ValueCached)
	assert(len(result.Value) == 1e+9)

}
func insert(c *Cache, key, size int, CacheUsed, ValueCached bool, Deleted int) {
	result := must(c.Get(context.Background(), fmt.Sprintf("%d:%d", key, size)))
	assert(result.CacheUsed == CacheUsed)
	assert(result.ValueCached == ValueCached)
	assert(result.Deleted == Deleted)
	assert(bytes.Equal(make([]byte, size), result.Value))
}
func TestCache_LRU(t *testing.T) {
	dbPath := filepath.Join(os.TempDir(), strconv.FormatInt(time.Now().Unix(), 10))
	db := must(leveldb.OpenFile(dbPath, &opt.Options{ErrorIfExist: true}))
	defer db.Close()
	defer os.RemoveAll(dbPath)
	cache := newCache(db, 1000, func(ctx context.Context, key string) ([]byte, error) {
		size := must(strconv.Atoi(strings.Split(key, ":")[1]))
		return make([]byte, size), nil
	})
	insert(cache, 1, 1000, false, false, 0)
	assert(cache.Size() == 0)
	insert(cache, 2, 999, false, true, 0)
	assert(cache.Size() == 999)
	insert(cache, 3, 555, false, true, 1)
	assert(cache.Size() == 555)
	insert(cache, 4, 400, false, true, 0)
	assert(cache.Size() == 555+400)
	insert(cache, 5, 1000, false, false, 0)
	assert(cache.Size() == 555+400)
	insert(cache, 6, 2000, false, false, 0)
	assert(cache.Size() == 555+400)
	insert(cache, 7, 999, false, true, 2)
	assert(cache.Size() == 999)
	insert(cache, 8, 100, false, true, 1)
	assert(cache.Size() == 100)
	insert(cache, 9, 100, false, true, 0)
	assert(cache.Size() == 2*100)
	insert(cache, 9, 100, true, true, 0)
	assert(cache.Size() == 2*100)
}
func TestCache_Size(t *testing.T) {
	dbPath := filepath.Join(os.TempDir(), strconv.FormatInt(time.Now().Unix(), 10))
	db := must(leveldb.OpenFile(dbPath, &opt.Options{ErrorIfExist: true}))
	defer db.Close()
	defer os.RemoveAll(dbPath)
	cache := newCache(db, 10000, func(ctx context.Context, key string) ([]byte, error) {
		return make([]byte, 100), nil
	})
	for i := range make([]struct{}, 111) {
		result := must(cache.Get(context.Background(), strconv.Itoa(i)))
		assert(!result.CacheUsed)
		assert(result.ValueCached)
		assert(result.Deleted == i/(101-1))
		assert(bytes.Equal(make([]byte, 100), result.Value))
	}
	assert(cache.Size() == 100*100)
}
func TestCache_Get(t *testing.T) {
	dbPath := filepath.Join(os.TempDir(), strconv.FormatInt(time.Now().Unix(), 10))
	db := must(leveldb.OpenFile(dbPath, &opt.Options{ErrorIfExist: true}))
	defer db.Close()
	defer os.RemoveAll(dbPath)
	cache := newCache(db, 10000, func(ctx context.Context, key string) ([]byte, error) {
		return []byte("bar"), nil
	})
	result := must(cache.Get(context.Background(), "foo"))
	if string(result.Value) != "bar" {
		t.Fatalf("expected bar get %s", string(result.Value))
	}
	if !result.ValueCached {
		t.Fatal("value not cached")
	}
	if result.CacheUsed {
		t.Fatal("CacheUsed is true")
	}
	if result.Deleted != 0 {
		t.Fatal("Delete is not zero")
	}
	result = must(cache.Get(context.Background(), "foo"))
	if string(result.Value) != "bar" {
		t.Fatalf("expected bar get %s", string(result.Value))
	}
	if !result.ValueCached {
		t.Fatal("value not cached")
	}
	if !result.CacheUsed {
		t.Fatal("cache not used")
	}
	if result.Deleted != 0 {
		t.Fatal("Delete is not zero")
	}
}
