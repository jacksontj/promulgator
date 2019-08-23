/**
* Copyright 2018 Comcast Cable Communications Management, LLC
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
* http://www.apache.org/licenses/LICENSE-2.0
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package tcache

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/model"
)

// ReapSleepMS should be configurable later
var ReapSleepMS int64 = 1000

// MemoryCache defines a a Memory Cache client that conforms to the Cache interface
type MemoryCache struct {
	T      *TCacheAPI
	client sync.Map
}

// CacheObject represents a Cached object as stored in the Memory Cache
type CacheObject struct {
	Key        string
	Value      model.Value
	Expiration int64
}

// Connect initializes the MemoryCache
func (c *MemoryCache) Connect() error {
	level.Info(c.T.Logger).Log("event", "memorycache setup")
	c.client = sync.Map{}
	go c.Reap()
	return nil
}

// Store places an object in the cache using the specified key and ttl
func (c *MemoryCache) Store(cacheKey string, data model.Value, ttl int64) error {
	level.Debug(c.T.Logger).Log("event", "memorycache cache store", "key", cacheKey)
	c.client.Store(cacheKey, CacheObject{Key: cacheKey, Value: data, Expiration: time.Now().Unix() + ttl})
	return nil
}

// Retrieve looks for an object in cache and returns it (or an error if not found)
func (c *MemoryCache) Retrieve(cacheKey string) (model.Value, error) {
	record, ok := c.client.Load(cacheKey)
	if ok {
		return record.(CacheObject).Value, nil
	}
	return nil, fmt.Errorf("Value  for key [%s] not in cache", cacheKey)
}

// Reap continually iterates through the cache to find expired elements and removes them
func (c *MemoryCache) Reap() {
	for {
		c.ReapOnce()
		time.Sleep(time.Duration(ReapSleepMS) * time.Millisecond)
	}
}

// ReapOnce makes a single iteration through the cache to to find and remove expired elements
func (c *MemoryCache) ReapOnce() {
	now := time.Now().Unix()

	c.client.Range(func(k, value interface{}) bool {
		if value.(CacheObject).Expiration < now {
			key := k.(string)
			level.Debug(c.T.Logger).Log("event", "memorycache cache reap", "key", key)

			c.T.ChannelCreateMtx.Lock()
			c.client.Delete(k)

			// Close out the channel if it exists
			if _, ok := c.T.ResponseChannels[key]; ok {
				close(c.T.ResponseChannels[key])
				delete(c.T.ResponseChannels, key)
			}

			c.T.ChannelCreateMtx.Unlock()
		}
		return true
	})
}

// Close is not used for MemoryCache, and is here to fully prototype the Cache Interface
func (c *MemoryCache) Close() error {
	return nil
}
