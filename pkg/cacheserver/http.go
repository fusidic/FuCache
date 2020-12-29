package cacheserver

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/fusidic/FuCache/pkg/consistenthash"
	"github.com/fusidic/FuCache/pkg/groupcache"
)

const (
	defaultBasePath = "/_groupcache"
	defaultReplicas = 50
)

// Pool implements PeerPicker for a pool of HTTP peers.
// self 用作记录自己的地址，包括主机名和端口
// basePath 作为节点间通讯地址的前缀，如 http://example.com/_groupcache/ 开头的请求
//   即用于节点间的访问
type Pool struct {
	// peer's base URL, e.g. "https://example.net:8000"
	self     string
	basePath string
	mu       sync.Mutex
	// consistenthash 中关于 hash、寻址的实现
	peers *consistenthash.Map
	// 各节点名:地址
	httpGetter map[string]*httpGetter
}

// NewPool initializes an HTTP pool of peers.
func NewPool(self string) *Pool {
	return &Pool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log inof with server name
func (p *Pool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP handle all http requests
func (p *Pool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basePath>/<groupName>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.basePath)+1:], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// log.Printf("[parse path] 1:%v 2:%v", parts[1], parts[2])
	groupName := parts[0]
	key := parts[1]
	group := groupcache.GetGroup(groupName)
	if group == nil {
		errorS := "no such group" + groupName
		http.Error(w, errorS, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

type httpGetter struct {
	// baseURL 为节点地址
	baseURL string
}

// Get implements method Get in interface grouphttp.PeerGetter
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

var _ groupcache.PeerGetter = (*httpGetter)(nil)

// Set updates the pool's list of peers(expect host addresses), which implements peers.PeerPicker interface.
func (p *Pool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetter = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetter[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer picks a peer according to key.
func (p *Pool) PickPeer(key string) (groupcache.PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetter[peer], true
	}
	return nil, false
}

// 仅传方法过去
var _ groupcache.PeerPicker = (*Pool)(nil)
