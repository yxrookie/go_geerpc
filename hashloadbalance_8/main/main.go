package main

import (
	"context"
	"errors"
	"net/http"
	
	hashloadbalance_8 "go_geerpc/hashloadbalance_8"
	"go_geerpc/hashloadbalance_8/breaker"
	"go_geerpc/hashloadbalance_8/registry"
	"go_geerpc/hashloadbalance_8/xclient"
	"log"
	"net"
	"sync"
	"time"
)
type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func (f Foo) Sleep(args Args, reply *int) error {
	time.Sleep(time.Second * time.Duration(args.Num1))
	*reply = args.Num1 + args.Num2
	return nil
}

func startRegistry(wg *sync.WaitGroup) {
	l, _ := net.Listen("tcp", ":9999")
	registry.HandleHTTP()
	wg.Done()
	_ = http.Serve(l, nil)
}

func startServer(registryAddr string, wg *sync.WaitGroup) {
	var foo Foo
	l, _ := net.Listen("tcp", ":0")
	server := hashloadbalance_8.NewServer()
	_ = server.Register(&foo)
	registry.Heartbeat(registryAddr, "tcp@"+l.Addr().String(), 0)
	wg.Done()
	server.Accept(l)
}

func foo(xc *xclient.XClient, ctx context.Context, typ, serviceMethod string, args *Args) error{
	var reply int
	var err error
	switch typ {
	case "call":
		err = xc.Call(ctx, serviceMethod, args, &reply)
	case "broadcast":
		err = xc.Broadcast(ctx, serviceMethod, args, &reply)
	}
	if err != nil {
		//log.Printf("%s %s error: %v", typ, serviceMethod, err)
		return errors.New("failed")
	} else {
		log.Printf("%s %s success: %d + %d = %d", typ, serviceMethod, args.Num1, args.Num2, reply)
	}
	return nil
}



func call(registry string) {
	d := xclient.NewGeeRegistryDiscovery(registry, 0)
	//xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	loadMode :=  xclient.HashConsistentSelect
	xc := xclient.NewXClient(d, loadMode, nil)

	if loadMode == xclient.HashConsistentSelect {
		// 初始化一个 hash 环
	d.Rb = xclient.NewHashBanlance(100, nil)
	servers, err := xc.D.GetAll()
	if err != nil {
		panic("hash 环添加节点失败！！！")
	}
	for _, server := range servers {
		d.Rb.Add(server)
	}
	}

	defer func() { _ = xc.Close() }()
	// send request & receive response
	var wg sync.WaitGroup

	foobraker := breaker.FooError{}


	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := foo(xc, context.Background(), "call", "Foo.Sum", &Args{Num1: i, Num2: i * i})
			foobraker.FooDoWithFallBack(err)
		}(i)
	}
	wg.Wait()
}


func broadcast(registry string) {
	d := xclient.NewGeeRegistryDiscovery(registry, 0)
	loadMode :=  xclient.HashConsistentSelect
	//xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	xc := xclient.NewXClient(d, loadMode, nil)

	if loadMode == xclient.HashConsistentSelect {
		// 初始化一个 hash 环
	d.Rb = xclient.NewHashBanlance(100, nil)
	servers, err := xc.D.GetAll()
	if err != nil {
		panic("hash 环添加节点失败！！！")
	}
	for _, server := range servers {
		d.Rb.Add(server)
	}
	}

	defer func() { _ = xc.Close() }()
	var wg sync.WaitGroup

	foobraker := breaker.FooError{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "broadcast", "Foo.Sum", &Args{Num1: i, Num2: i * i})
			// expect 2 - 5 timeout
			ctx, _ := context.WithTimeout(context.Background(), time.Second*20)
			err := foo(xc, ctx, "broadcast", "Foo.Sleep", &Args{Num1: i, Num2: i * i})
			foobraker.FooDoWithFallBack(err)
		}(i)
	}
	wg.Wait()
	
}

func main() {
	log.SetFlags(0)
	registryAddr := "http://localhost:9999/_geerpc_/registry"
	var wg sync.WaitGroup
	wg.Add(1)
	go startRegistry(&wg)
	wg.Wait()

	time.Sleep(time.Second)
	wg.Add(2)
	go startServer(registryAddr, &wg)
	go startServer(registryAddr, &wg)
	wg.Wait()

	time.Sleep(time.Second)
	call(registryAddr)
	broadcast(registryAddr)
}