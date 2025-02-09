package groupcache

import (
	"fmt"
	"log"
	"sync"

	"github.com/fusidic/FuCache/pkg/singleflight"
	"github.com/fusidic/FuCache/proto/cachepb"
)

// Getter is a interface to get data stored in cache,
// it contains a method Get, which should be implemented by user.
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc implements Getter.
type GetterFunc func(key string) ([]byte, error)

// Get implements Getter.Get()
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// Group is a cache namespace and associate data in all nodes.
// Group 是缓存的命名空间，每个 Group 拥有唯一 name，如可以创建三个 Group：
//   学生成绩 scores，学生信息 info，学生课程 courses
// getter 为当未命中时获取源数据的 callback
// mainCache 并发缓存 (cache.go)
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
	// use singleflight.Group to make sure that each key is only fetched once
	loader *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup create a new instance of Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// 核心方法实现

// Get value for a key from cache
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("Require a key")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Printf("[GroupCache] hit")
		return v, nil
	}
	return g.load(key)
}

// RegisterPeers registers a PeerPicker for choosing remote peer.
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// load value if not exist
// 单机环境下，会从数据源中回调；分布式环境下，会从其他节点中回调
func (g *Group) load(key string) (value ByteView, err error) {
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			// 根据哈希，选择远程节点
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GroupCache] Failed to get from peer", err)
			}
		}

		// 此处逻辑感觉有些不对
		return g.getLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &cachepb.Request{
		Group: g.name,
		Key:   key,
	}

	res := &cachepb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		// log.Printf("[Server] Not exist, loading ...")
		// g.getLocally(key)
		// bytes, _ := peer.Get(g.name, key)
		// return ByteView{b: bytes}, nil
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}

func (g *Group) getLocally(key string) (ByteView, error) {
	// 调用 getter.Get 获取数据源
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	// 添加到缓存中
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
