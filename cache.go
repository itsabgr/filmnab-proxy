package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"sync"
	"sync/atomic"
	"time"
)

type indexEntry struct {
	lastRead  int64
	valueSize int64
}
type index struct {
	map_ sync.Map
	size atomic.Int64
}

func (i *index) sumSizes() int64 {
	sum := i.size.Load()
	assert(sum >= 0)
	return sum
}
func (i *index) delete(key string) {
	value, ok := i.map_.LoadAndDelete(key)
	if ok {
		i.size.Add(-value.(indexEntry).valueSize)
	}
}
func (i *index) leastRead() (key string) {
	least := struct {
		indexEntry
		key string
		any bool
	}{}
	least.lastRead = ^int64(0)
	i.map_.Range(func(key, value any) bool {
		entry := value.(indexEntry)
		str := key.(string)
		least.any = true
		if entry.lastRead < least.lastRead || (entry.lastRead == entry.lastRead && entry.valueSize > least.valueSize) {
			least.lastRead = entry.lastRead
			least.valueSize = entry.valueSize
			least.key = str
		}
		return true
	})
	if !least.any {
		return ""
	}
	return least.key
}
func (i *index) reset(key string, size int64) {
	pre, ok := i.map_.Swap(key, indexEntry{
		lastRead:  time.Now().UTC().Unix(),
		valueSize: size,
	})
	if ok {
		i.size.Add(size - pre.(indexEntry).valueSize)
	} else {
		i.size.Add(size)
	}
}

type OnMissing func(ctx context.Context, key string) ([]byte, error)
type Cache struct {
	db        *leveldb.DB
	index     index
	max       int64
	onMissing OnMissing
}

func newCache(db *leveldb.DB, max int64, onMissing OnMissing) *Cache {
	return &Cache{
		db:        db,
		onMissing: onMissing,
		max:       max,
	}
}
func (c *Cache) find(ctx context.Context, key string) (val []byte, cached bool, countDeleted int, err error) {
	val, err = c.onMissing(ctx, key)
	if err != nil {
		return nil, false, 0, err
	}
	if len(val) == 0 {
		return nil, false, 0, nil
	}
	ok, count, err := c.clean(int64(len(val)))
	if err != nil || !ok {
		return val, false, count, err
	}
	if c.db.Put([]byte(key), val, nil) != nil {
		return val, false, count, err
	}
	c.index.reset(key, int64(len(val)))
	return val, true, count, nil
}
func (c *Cache) clean(val int64) (bool, int, error) {
	if val >= c.max {
		return false, 0, nil
	}
	n := 0
	for c.Size()+val > c.max {
		least := c.index.leastRead()
		if least == "" {
			panic(errors.New("unreachable"))
		}
		if err := c.db.Delete([]byte(least), nil); err != nil {
			return false, n, err
		}
		c.index.delete(least)
		n += 1
	}
	return true, n, nil
}

type result struct {
	CacheUsed   bool   `json:"cached"`
	ValueCached bool   `json:"stored"`
	Deleted     int    `json:"deleted"`
	Value       []byte `json:"-"`
}

func (r *result) Header() string {
	return fmt.Sprintf("%t,%t,%d", r.CacheUsed, r.ValueCached, r.Deleted)
}
func (c *Cache) Get(ctx context.Context, key string) (result, error) {
	if key == "" {
		panic(errors.New("empty key"))
	}
	val, err := c.db.Get([]byte(key), nil)
	switch err {
	case leveldb.ErrNotFound:
		val, cached, countDeleted, err := c.find(ctx, key)
		return result{false, cached, countDeleted, val}, err
	case nil:
		c.index.reset(key, int64(len(val)))
		return result{true, true, 0, val}, err
	default:
		panic(err)
	}
}
func (c *Cache) Size() int64 {
	return c.index.sumSizes()
}
