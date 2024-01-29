package main

import (
	"context"
	"errors"
	"net/http"

	circuitbreaker_7 "go_geerpc/circuitbreaker_7"
	"go_geerpc/circuitbreaker_7/registry"
	"go_geerpc/circuitbreaker_7/xclient"
	"go_geerpc/circuitbreaker_7/breaker"
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
	server := circuitbreaker_7.NewServer()
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
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()
	// send request & receive response
	var wg sync.WaitGroup

	foobraker := breaker.FooError{}
	for i := 0; i < 10; i++ {
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
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()
	var wg sync.WaitGroup

	foobraker := breaker.FooError{}
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "broadcast", "Foo.Sum", &Args{Num1: i, Num2: i * i})
			// expect 2 - 5 timeout
			ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
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
	//call(registryAddr)
	broadcast(registryAddr)
}