package xclient

import (
	"context"
	"fmt"
	. "go_geerpc/hashloadbalance_8"
	"io"
	"math/rand"
	"reflect"
	"strings"
	"sync"
)

type XClient struct {
	D Discovery
	mode SelectMode
	opt *Option
	mu sync.Mutex
	clients map[string]*Client
}

var _ io.Closer = (*XClient) (nil)

func NewXClient(d Discovery, mode SelectMode, opt *Option) *XClient {
	return &XClient{D: d, mode: mode, opt: opt, clients: make(map[string]*Client)}
	
}

func(xc *XClient) Close() error {
	xc.mu.Lock()
	defer xc.mu.Unlock()
	for key, client := range xc.clients {
		_ = client.Close()
		delete(xc.clients, key)
	}
	return nil
}

func (xc *XClient) dial (rpcAddr string) (*Client, error) {
	xc.mu.Lock()
	defer xc.mu.Unlock()
	client, ok := xc.clients[rpcAddr]
	if ok && !client.IsAvailable() {
		_ = client.Close()
		delete(xc.clients, rpcAddr)
		client = nil
	}
	if client == nil {
		parts := strings.Split(rpcAddr, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("rpc client err: wrong format '%s', expect protocol@addr", rpcAddr)
		} 
		protocol, addr := parts[0], parts[1]
		// bug: can't use := here
		//client, err := Dial(protocol, addr, xc.opt)
		var err error
		client, err = Dial(protocol, addr, xc.opt)
		if client == nil {
			err =  fmt.Errorf("client is nil")
			return nil, err
		}
		if err != nil {
			return nil, err
		}
		xc.clients[rpcAddr] = client
	}
	return client, nil
}

func (xc *XClient) call(rpcAddr string, ctx context.Context, serviceMethod string, args, reply interface{}) error {
	client, err := xc.dial(rpcAddr)
	
	if err != nil {
		return err
	}
	
	return client.Call(ctx, serviceMethod, args, reply)
}

// 定义随机字符串的字符集
const charSet string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// generateRandomString 生成指定长度的随机字符串
func generateRandomString(charSet string, length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charSet[rand.Intn(len(charSet))]
	}
	return string(b)
}

// Call invokes the named function, waits for it to complete,
// and returns its error status.
// xc will choose a proper server.
func (xc *XClient) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	rpcAddr, err := xc.D.Get(xc.mode, generateRandomString(charSet, 10))
	//rpcAddr, err := xc.D.Get(xc.mode, serviceMethod)
	if err != nil {
		return err
	}
	return xc.call(rpcAddr, ctx, serviceMethod, args, reply)
}

//调用的 Broadcast 方法和三种负载均衡策略无关
//Broadcast 将请求广播到所有的服务实例，如果任意一个实例发生错误，则返回其中一个错误
func (xc *XClient) Broadcast(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	servers, err := xc.D.GetAll()
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var e error
	replyDone := reply == nil 
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for _, rpcAddr := range servers {
		wg.Add(1)
		go func(rpcAddr string) {
			defer wg.Done()
			var clonedReply interface{}
			if reply != nil {
				clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
			}
			err := xc.call(rpcAddr, ctx, serviceMethod, args, clonedReply)
			mu.Lock()
			if err != nil && e == nil {
				e = err
				cancel()
			}
			if err == nil && !replyDone {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonedReply).Elem())
				replyDone = true
			}
			mu.Unlock()
		}(rpcAddr)
	}
	wg.Wait()
	return e
}