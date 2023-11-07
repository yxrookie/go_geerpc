package xclient

import (
	"context"
	
	"math"
	"math/rand"
	"reflect"
	"sync"
	"time"
)

type SelectMode int

const (
	//RandomSelect     SelectMode = iota // select randomly
	//RoundRobinSelect                   // select using Robbin algorithm
	ConsistentHash 	SelectMode = iota
)

type Discovery interface {
	Refresh() error
	Update(servers []string) error
	//Get(mode SelectMode) (string, error)
	GetAll() ([]string, error)
	GetConsisHash(mode SelectMode, serviceMethod string) (string, error)
}

// MulitServersDiscovery is a discovery for multi servers without a registry center
// user provides the server addresses explicity instead
type MulitServersDiscovery struct {
	r *rand.Rand  // generate random number
	mu sync.RWMutex // protect following
	servers []string
	index int  // record the selected position for robin algorithm

	rb *HashBanlance
}

// NewMultiServerDiscovery creates a MultiServersDiscovery instance
func NewMultiServerDiscovery(servers []string) *MulitServersDiscovery {
	d := &MulitServersDiscovery{
		servers: servers,
		r: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	d.index = d.r.Intn(math.MaxInt32 - 1)
    
	// 初始化一个 hash 环
	d.rb = NewHashBanlance(10, nil)
	for _, server := range servers {
		d.rb.Add(server)
	}
	return d
}

var _ Discovery = (*MulitServersDiscovery)(nil)

// Refresh doesn't make sense for MultiServersDiscovery, so ignore it
func (d *MulitServersDiscovery) Refresh() error {
	return nil
}

// Update the servers of discovery dynamically if needed 
func (d *MulitServersDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	return nil
}
/* 
// Get a server according to mode
func (d *MulitServersDiscovery) Get(mode SelectMode) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := len(d.servers)
	if n == 0 {
		return "", errors.New("rpc discovery: no available servers")
	}
	switch mode {
	 case RandomSelect:
		return d.servers[d.r.Intn(n)], nil
	case RoundRobinSelect:
		s := d.servers[d.index%n]  //servers could be updated, so mode n to ensure safety 
		d.index = (d.index + 1) % n
		return s, nil 
	default:
		return "", errors.New("rpc discovery: not supported select mode")
	}
}
 */
func (d *MulitServersDiscovery) GetConsisHash(mode SelectMode, serviceMethod string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.rb.Get(serviceMethod)
}


// returns all servers in discovery
func (d *MulitServersDiscovery) GetAll() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	// return a copy of d.servers
	servers := make([]string, len(d.servers), len(d.servers))
	copy(servers, d.servers)
	return servers, nil
}

func (xc *XClient) Broadcast(ctx  context.Context, serviceMethod string, args, reply interface{}) error {
	servers, err := xc.d.GetAll()
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var mu sync.Mutex   // protect e and replyDone
	var e error
	replyDone := reply == nil 
	ctx, cancel := context.WithCancel(ctx)
	for _, rpcAddr := range servers {
		wg.Add(1)
		go func(rpcAddr string) {
			defer wg.Done()
			var cloneReply interface{}
			if reply != nil {
				cloneReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
			}
			err := xc.call(rpcAddr, ctx, serviceMethod, args, cloneReply)
			mu.Lock()
			if err != nil && e == nil {
				e = err
				cancel()
			}
			if err == nil && !replyDone {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(cloneReply).Elem())
				replyDone = true
			}
			mu.Unlock()
		}(rpcAddr)
	}
	wg.Wait()
	return e
}

