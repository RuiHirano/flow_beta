package main

import (
	"fmt"
)

func GetCallback(cb CallbackInterface) func() {
	newCb := NewBaseCallback(cb)
	return newCb.AgentCallback
}

type CallbackInterface interface {
	SetAgentRequest()
	SetAgentResponse()
}

type Callback struct {
	Name string
}

func NewCallback(name string) *Callback {
	cb := &Callback{
		Name: name,
	}
	return cb
}

func (cb *Callback) SetAgentRequest() {
	fmt.Printf("request1")
}

func (cb *Callback) SetAgentResponse() {
	fmt.Printf("response1 %s", cb.Name)
}

//////////////////////////////////////////////////////////////
type BaseCallback struct {
	Id string
	CallbackInterface
}

func NewBaseCallback(bc CallbackInterface) *BaseCallback {
	cb2 := &BaseCallback{
		Id:                "aaa",
		CallbackInterface: bc,
	}
	return cb2
}

func (cb2 *BaseCallback) AgentCallback() {
	cb2.SetAgentRequest()
	cb2.SetAgentResponse()
}
