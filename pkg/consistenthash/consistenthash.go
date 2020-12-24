package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32

// Map contains all hashed keys
type Map struct {
	hash     Hash
	replicas int            // 虚拟节点倍数
	keys     []int          // Sorted 哈希环，放置虚拟节点的哈希值
	hashMap  map[int]string // k:虚拟节点的哈希值 v:真实节点的名称
}

// New creates a Map instance
// Hash will be set as crc32.ChecksumIEEE by default.
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add adds some keys to the hash.
// Add 允许传入0或多个真实节点的名称，对应每个真实节点，创建 m.replicas 个虚拟节点
// 虚拟节点的名称为 strconv.Itoa(i) + key
// 使用 m.hash() 将虚拟节点映射到环 keys 上
// 在 hashMap 中维护虚拟节点与真实节点之间的映射关系
// 最后会将环 keys 上的哈希值进行排序 (从小到大)
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			// 专程字节码并取哈希
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// Get gets the closest item in the hash to the provided key.
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	// 根据 key 计算哈希值
	hash := int(m.hash([]byte(key)))
	// sort.Search 返回第一个为 true 的值，否则返回 n，这里用于寻找虚拟节点
	idx := sort.Search(len(m.keys), func(i int) bool {
		// 首个大于 hash 的哈希值，即虚拟节点
		return m.keys[i] >= hash
	})

	// 获取真实节点的名称
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
