package groupcache

import "github.com/fusidic/FuCache/proto/cachepb"

// PeerPicker is the interface that must be implemented to
// locate the peer that owns a specific key
type PeerPicker interface {
	// PickPeer 根据传入的 Key 选择对应的节点进行 PeerGetter 操作
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter is the interface that must be implemented by a peer.
type PeerGetter interface {
	// 从对应 group 中查找缓存值
	// Get(group string, key string) ([]byte, error)
	// protobuf
	Get(in *cachepb.Request, out *cachepb.Response) error
}
