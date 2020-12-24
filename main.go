package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/fusidic/FuCache/pkg/cacheclient"
	"github.com/fusidic/FuCache/pkg/memcache"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	memcache.NewGroup("scores", 2<<10, memcache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	addr := "localhost:9999"
	peers := cacheclient.NewPool(addr)
	log.Println("memcache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}
