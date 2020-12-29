package groupcache

import (
	"fmt"
	"log"
	"sync"
	"testing"
)

func Test_get(t *testing.T) {
	cache := &cache{
		mu:         sync.Mutex{},
		lru:        nil,
		cacheBytes: 10,
	}
	var r bool
	v, ok := cache.get("key")
	log.Printf("value: '%v' ok: %v r: %v\n", v, ok, r)
	fmt.Printf("%v %v\n", v, ok)
}
