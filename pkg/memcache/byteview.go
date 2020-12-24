package memcache

// ByteView holds an immutable view of bytes.
// 提供字节形式的存储，可以兼容多种数据源（文本、图片等）
type ByteView struct {
	b []byte
}

// Len implements interface lru.Value.Len()
func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice returns a copy of the data as a byte slice, incase it will be modified.
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
