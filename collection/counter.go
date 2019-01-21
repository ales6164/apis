package collection

import (
	"fmt"
	"math/rand"

	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/memcache"
)

type counterConfig struct {
	Shards int
}

type shard struct {
	Name  string
	Count int
}

const (
	defaultShards = 20
	configKind    = "_counterShardConfig"
	shardKind     = "_counterShard"
)

func memcacheKey(name string) string {
	return shardKind + ":" + name
}

// Count retrieves the value of the named counter.
func (c *Collection) Count(ctx context.Context) (int, error) {
	total := 0
	mkey := memcacheKey(c.name)
	if _, err := memcache.JSON.Get(ctx, mkey, &total); err == nil {
		return total, nil
	}
	q := datastore.NewQuery(shardKind).Filter("Name =", c.name)
	for t := q.Run(ctx); ; {
		var s shard
		_, err := t.Next(&s)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return total, err
		}
		total += s.Count
	}
	_ = memcache.JSON.Set(ctx, &memcache.Item{
		Key:        mkey,
		Object:     &total,
		Expiration: 60,
	})
	return total, nil
}

// Increment increments the named counter.
func (c *Collection) Increment(ctx context.Context) error {
	// Get counter config.
	var cfg counterConfig
	ckey := datastore.NewKey(ctx, configKind, c.name, 0, nil)
	err := datastore.Get(ctx, ckey, &cfg)
	if err == datastore.ErrNoSuchEntity {
		cfg.Shards = defaultShards
		_, err = datastore.Put(ctx, ckey, &cfg)
	}

	if err != nil {
		return err
	}
	var s shard
	shardName := fmt.Sprintf("%s-shard%d", c.name, rand.Intn(cfg.Shards))
	key := datastore.NewKey(ctx, shardKind, shardName, 0, nil)
	err = datastore.Get(ctx, key, &s)
	// A missing entity and a present entity will both work.
	if err != nil && err != datastore.ErrNoSuchEntity {
		return err
	}
	s.Name = c.name
	s.Count++
	_, err = datastore.Put(ctx, key, &s)
	if err != nil {
		return err
	}
	_, _ = memcache.IncrementExisting(ctx, memcacheKey(c.name), 1)
	return nil
}

// Decrement decrements the named counter.
func (c *Collection) Decrement(ctx context.Context) error {
	// Get counter config.
	var cfg counterConfig
	ckey := datastore.NewKey(ctx, configKind, c.name, 0, nil)
	err := datastore.Get(ctx, ckey, &cfg)
	if err == datastore.ErrNoSuchEntity {
		cfg.Shards = defaultShards
		_, err = datastore.Put(ctx, ckey, &cfg)
	}
	if err != nil {
		return err
	}
	var s shard

	shardName := fmt.Sprintf("%s-shard%d", c.name, rand.Intn(cfg.Shards))
	key := datastore.NewKey(ctx, shardKind, shardName, 0, nil)
	err = datastore.Get(ctx, key, &s)
	// A missing entity and a present entity will both work.
	if err != nil && err != datastore.ErrNoSuchEntity {
		return err
	}
	s.Name = c.name
	s.Count--
	_, err = datastore.Put(ctx, key, &s)
	if err != nil {
		return err
	}
	_, _ = memcache.IncrementExisting(ctx, memcacheKey(c.name), 1)
	return nil
}
