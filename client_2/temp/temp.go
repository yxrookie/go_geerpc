package main

import (
    "fmt"
    "time"
)

// Call represents a hypothetical asynchronous operation
type Call struct {
    // Done is a channel that signals completion of the operation
    Done chan *Call
}

// done signals the completion of the Call
func (call *Call) done() {
    call.Done <- call
}

// simulateAsyncOperation simulates an asynchronous operation
func simulateAsyncOperation(call *Call) {
    // Simulate some work with a sleep
    time.Sleep(2 * time.Second)
    // Signal that the work is done
    call.done()
}

func main() {
    // Create an instance of Call with a buffered Done channel
    call := &Call{Done: make(chan *Call, 1)}

    // Start the asynchronous operation
    go simulateAsyncOperation(call)
    // Wait for the operation to complete
    completedCall := <-call.Done
    fmt.Println("Operation completed:", completedCall)
}
