package main

import (
	"fmt"
	
	"reflect"
	
)

type Mystruct struct {} 

func main() {
	mystruct := &Mystruct{}
	rcvr := reflect.ValueOf(mystruct)
	
	nameWithIndirect := reflect.Indirect(rcvr).Type().Kind()
	nameWithoutdirect := rcvr.Type().Kind()
	//struct
	//ptr
	fmt.Println(nameWithIndirect)
	fmt.Println(nameWithoutdirect)
}
