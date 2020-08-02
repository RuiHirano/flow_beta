package main

import "fmt"

type Callback struct {
	*BaseCallback
}

func (cb *Callback) SetAgentRequest() {
	fmt.Printf("request2\n")
}

func (cb *Callback) SetAgentResponse() {
	fmt.Printf("response2 %s\n", cb.Name)
}

func main() {
	cb := NewBaseCallback("rui")
	cb2 := &Callback{cb}
	cb2.SetAgentRequest()
	cb2.SetAgentResponse()
}
