package main

import "fmt"

type MyCallback struct {
	*Callback
}

func (cb *MyCallback) SetAgentRequest() {
	fmt.Printf("request2\n")
}

func (cb *MyCallback) SetAgentResponse() {
	fmt.Printf("response2 %s\n", cb.Name)
}

func main() {
	cb := NewCallback("rui")
	mycb := &MyCallback{cb} // override
	callback := GetCallback(mycb)
	callback()
	// ここまではよい
	newCb := NewBaseCallback(mycb)
	newCb.AgentCallback()

	// Overrideしたものを与える
	//newCb := NewCallback2(cb2)
	//newCb.AgentCallback()
}
