package cacheclient

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/fusidic/FuCache/pkg/memcache"
)

const defaultBasePath = "/_memcache"

// Pool implements PeerPicker for a pool of HTTP peers.
// self 用作记录自己的地址，包括主机名和端口
// basePath 作为节点间通讯地址的前缀，如 http://example.com/_memcache/ 开头的请求
//   即用于节点间的访问
type Pool struct {
	// peer's base URL, e.g. "https://example.net:8000"
	self     string
	basePath string
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
	group := memcache.GetGroup(groupName)
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
