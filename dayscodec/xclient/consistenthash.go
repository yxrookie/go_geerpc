package xclient

import (
	"errors"
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

type Hash func(data []byte) uint32

type UInt32Slice []uint32

func (s UInt32Slice) Len() int {
	return len(s)
}

func (s UInt32Slice) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s UInt32Slice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type LoadBalanceConf interface {

}

type HashBanlance struct {
	mux 		sync.RWMutex
	hash 		Hash
	replicas 	int   //复制因子
	keys 		UInt32Slice  //已排序的节点 hash 切片
	hashMap 	map[uint32]string  //节点哈希和Key的map，键是 hash 值，值是节点 key

	//观察主体
	conf LoadBalanceConf
}

func NewHashBanlance(replicas int, fn Hash) *HashBanlance {
	m := &HashBanlance{
		replicas: replicas,
		hash: fn,
		hashMap: make(map[uint32]string),
	}
	if m.hash == nil {
        // 最多 32 位，保证是一个2^32-1的环，对于输入相同的数据，则产生的结果一样，数据不同，产生的结果也不同
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 验证是否为空
func(c *HashBanlance) IsEmpty() bool {
	return len(c.keys) == 0
}

// Add 方法用来添加缓存节点，参数节点为 key
func (c *HashBanlance) Add(params ...string) error {
	if len(params) == 0 {
		return errors.New("param len 1 at least")
	}
	addr := params[0]
	c.mux.Lock()
	defer c.mux.Unlock()
	for i := 0; i < c.replicas; i++ {
		hash := c.hash([]byte(strconv.Itoa(i)+addr))
		c.keys = append(c.keys, hash)
		c.hashMap[hash] = addr
	}
	sort.Sort(c.keys)
	return nil
}

// 查询所有节点
func (c *HashBanlance) countNode() {
	c.mux.RLock()
	defer c.mux.RUnlock()
	fmt.Println("当前节点总数为 ", len(c.keys))
	fmt.Println(c.keys)
}

// 删除缓存节点
func (c *HashBanlance) Remove(params ...string) error {
	if len(params) == 0 {
		return errors.New("param len 1 at least")
	}
	addr := params[0]
	c.mux.Lock()
	defer c.mux.Unlock()
	for i := 0; i < c.replicas; i++ {
		nowhash := c.hash([]byte(strconv.Itoa(i)+addr))
		// 删除映射关系
		delete(c.hashMap, nowhash)
        if success, index := getIndexForKey(nowhash, c.keys); success {
			c.keys = append(c.keys[:index], c.keys[index+1:]...)
		}
	}
    return nil
}

// func getIndexForKey(nowhash Hash, keys UInt32Slice) (success bool, index int) {
func getIndexForKey(nowhash uint32, keys UInt32Slice) (success bool, index int) {
	for i, v := range keys {
		if v == nowhash {
			index = i
			return true, i
		}
	}
	return false, -1
}

func (c *HashBanlance) Get(key string) (string, error) {
	if c.IsEmpty() {
		return "", errors.New("node is empty")
	}
	hash := c.hash([]byte(key))

	idx := sort.Search(len(c.keys), func(i int) bool {return c.keys[i] >= hash})

	if idx == len(c.keys) {
		idx = 0
	}
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.hashMap[c.keys[idx]], nil
}

func (c *HashBanlance) SetConf(conf LoadBalanceConf) {
	c.conf = conf
}