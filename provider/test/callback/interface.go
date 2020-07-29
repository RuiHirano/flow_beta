package main

import (
	"fmt"
)

func RegisterCallback(cbif CallbackInterface) {

}

type CallbackInterface interface {
	SetAgentRequest()
	SetAgentResponse()
}

type BaseCallback struct {
	Name string
}

func NewBaseCallback(name string) *BaseCallback {
	cb := &BaseCallback{
		Name: name,
	}
	return cb
}

func (cb *BaseCallback) SetAgentRequest() {
	fmt.Printf("request1")
}

func (cb *BaseCallback) SetAgentResponse() {
	fmt.Printf("response1 %s", cb.Name)
}
