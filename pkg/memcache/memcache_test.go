package memcache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

// simulate database
var db = map[string]string{
	"Tom":     "630",
	"Jack":    "234",
	"fusidic": "556",
}

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))
	mem := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key]++
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	for k, v := range db {
		// 第一次读取
		if view, err := mem.Get(k); err != nil || view.String() != v {
			t.Fatalf("failed to get value of %s", k)
		}
		// 第二次读取
		if _, err := mem.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s missed", k)
		}
	}

	if view, err := mem.Get("unknown"); err == nil {
		t.Fatalf("the value of unknown should be empty, but %s got", view)
	} else {
		log.Printf("%s", err)
	}
}
