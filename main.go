package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/fusidic/FuCache/pkg/cacheserver"
	"github.com/fusidic/FuCache/pkg/groupcache"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *groupcache.Group {
	return groupcache.NewGroup("scores", 2<<10, groupcache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[mainDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// 开启本地节点服务，并将地址填入 Pool，注册到 Group 中
func startCacheServer(addr string, addrs []string, group *groupcache.Group) {
	node := cacheserver.NewPool(addr)
	node.Set(addrs...)
	// Pool 中有 PickPeer 实现
	group.RegisterPeers(node)
	log.Println("groupcache is running at ", addr)
	log.Fatal(http.ListenAndServe(addr[7:], node))
}

// 用户访问端口
func startAPIServer(apiAddr string, group *groupcache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := group.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))
	log.Println("frontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Groupcache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	// Peer nodes
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	group := createGroup()

	// 只有当输入参数 api 为 true 时，才会开启唯一的 API Server
	if api {
		go startAPIServer(apiAddr, group)
	}
	startCacheServer(addrMap[port], []string(addrs), group)
}
