package main

import (
	"fmt"
	"sync"

	
)

type Mystruct struct {} 

func main() {
	myPool := &sync.Pool{
		New: func() interface{} {
			fmt.Println("Creating new instance.")
			return struct{}{}
		},
	}
	myPool.Get()
	instance := myPool.Get()
	myPool.Put(instance)
	myPool.Get()
}
