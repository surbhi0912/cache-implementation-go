package main

//TODO : cache struct, janitor struct, using ticker, thread safe map, lock and rlock, expiration, purging

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

const (
	DefaultExpiration  time.Duration = 0
	InfiniteExpiration time.Duration = -1
)

type Item struct {
	itemValue      interface{}
	itemExpiryTime int64
}

type cache struct {
	cacheExpirationDuration time.Duration
	hashTable               map[string]Item
	mutexLock               sync.RWMutex
	janitorToClean          *Janitor
}

type Janitor struct {
	evictionInterval time.Duration
	stop             chan bool
}

type Cache struct {
	*cache
}

func (c *cache) Set(key string, x interface{}, expirationDuration time.Duration) {
	if expirationDuration == DefaultExpiration { // 0 expiration duration set for the item
		expirationDuration = c.cacheExpirationDuration //expiration duration for item is set as cache expiration duration
	}
	var expirationTime int64
	if expirationDuration > 0 {
		expirationTime = time.Now().Add(expirationDuration).UnixNano()
	}
	c.mutexLock.Lock()
	defer c.mutexLock.Unlock()
	c.hashTable[key] = Item{
		itemValue:      x,
		itemExpiryTime: expirationTime,
	}
}

func (c *cache) Get(key string) (interface{}, bool) {
	c.mutexLock.RLock()
	defer c.mutexLock.RUnlock()
	item, found := c.hashTable[key]
	if found {
		if item.itemExpiryTime > time.Now().UnixNano() { //item not expired
			return item.itemValue, true
		} else { //item exists but expired
			return nil, false
		}
	} else {
		return nil, false
	}
}

func (c *cache) delete(key string) (interface{}, bool) {
	item, found := c.hashTable[key]
	if found {
		delete(c.hashTable, key)
		return item.itemValue, true
	}
	return nil, false
}

func (c *cache) Delete(key string) {
	c.mutexLock.Lock()
	defer c.mutexLock.Unlock()
	c.delete(key)
}

func (c *cache) removeExpired() {
	now := time.Now().UnixNano()
	for key, item := range c.hashTable {
		// fmt.Println(key, item, now, item.itemExpiryTime)
		if item.itemExpiryTime < now { //item has expired
			// fmt.Println("Removing expired")
			// fmt.Println(time.Now().UnixNano())
			c.delete(key)
		}
	}
}

func New(cacheExpirationDur time.Duration, evictionDur time.Duration) *Cache {
	if cacheExpirationDur == DefaultExpiration { //No expiration set
		cacheExpirationDur = InfiniteExpiration
	}

	c := &cache{
		cacheExpirationDuration: cacheExpirationDur,
		hashTable:               make(map[string]Item),
	}
	C := &Cache{c}
	if evictionDur > 0 {
		initiateCleanWithJanitor(c, evictionDur)
		runtime.SetFinalizer(C, stopCleaningWithJanitor)
	}
	return C
}

func initiateCleanWithJanitor(c *cache, evictionDur time.Duration) {
	j := &Janitor{
		evictionInterval: evictionDur,
		stop:             make(chan bool),
	}
	c.janitorToClean = j
	// fmt.Println("Janitor initialised")
	// fmt.Println(time.Now().UnixNano())
	go j.CleaningJanitor(c)
}

func (j *Janitor) CleaningJanitor(c *cache) {
	// fmt.Println("Janitor is cleaning")
	// fmt.Println(time.Now().UnixNano())
	ticker := time.NewTicker(j.evictionInterval)
	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-j.stop: //when stop channel returns true
			ticker.Stop()
		}
	}
}

func stopCleaningWithJanitor(C *Cache) {
	C.janitorToClean.stop <- true
}

func main() {
	fmt.Println(time.Now().UnixNano())
	fmt.Println(5 * time.Minute)
	fmt.Println(time.Now().UnixNano())
	fmt.Println(time.Now().Add(5 * time.Minute).UnixNano())
	fmt.Println("#########")

	c := New(3*time.Minute, 1*time.Minute)
	fmt.Println(c.cacheExpirationDuration)
	fmt.Println(c.hashTable)
	c.Set("aa", "bcdf", 40*time.Second)
	c.Set("bb", "ghi", 50*time.Second)
	c.Set("cc", "jhgt", 90*time.Second)
	c.Set("dd", "mnop", 0)
	c.Set("ee", "xyzw", -1)
	fmt.Println(c.hashTable)
	x, found := c.Get("aa")
	if found {
		fmt.Println("Value found at key aa", x)
	}

	time.Sleep(20 * time.Second)
	fmt.Println("After waiting 20s, seeing what items expired and removed")
	fmt.Println(c.hashTable)
	x, found = c.Get("aa")
	if found {
		fmt.Println("Value found at key aa", x)
	}
	x, found = c.Get("cc")
	if found {
		fmt.Println("Value found at key cc", x)
	}

	time.Sleep(1 * time.Minute)
	fmt.Println("After waiting 1m, seeing what items expired and removed")
	fmt.Println(c.hashTable)

	time.Sleep(1 * time.Minute)
	fmt.Println("After waiting 2m, seeing what items expired and removed")
	fmt.Println(c.hashTable)
}
