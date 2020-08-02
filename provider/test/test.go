package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"log"
	"sync"
	"time"

	api "github.com/RuiHirano/flow_beta/api"
	util "github.com/RuiHirano/flow_beta/util"
	sxapi "github.com/synerex/synerex_api"
	sxutil "github.com/synerex/synerex_sxutil"
)

var (
	sclientOpts map[uint32]*util.SclientOpt
	simapi      *api.SimAPI
)

func init() {
	sclientOpts = map[uint32]*util.SclientOpt{
		uint32(api.ChannelType_CLOCK): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_CLOCK),
			MBusCallback: MbcbClock,
			ArgJson:      fmt.Sprintf("{Client:TestProvider_Clock}"),
		},
		uint32(api.ChannelType_PROVIDER): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_PROVIDER),
			MBusCallback: MbcbProvider,
			ArgJson:      fmt.Sprintf("{Client:TestProvider_Provider}"),
		},
	}
	uid, _ := uuid.NewRandom()
	myProvider := &api.Provider{
		Id:   uint64(uid.ID()),
		Name: "TestProvider",
		Type: api.Provider_MASTER,
	}
	simapi = api.NewSimAPI(myProvider)
	log.Printf("ProviderID: %d", simapi.Provider.Id)
}

func MbcbClock(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got clock callback")
}

func MbcbProvider(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got provider callback")
}

func SendClockMsg() {
	clockSclient := sclientOpts[uint32(api.ChannelType_CLOCK)].Sclient
	ctx := context.Background()
	msg := &sxapi.MbusMsg{
		ArgJson: "test",
	}
	log.Printf("Send Clock Info %d", msg.GetTargetId())
	clockSclient.SendMbusMsg(ctx, msg)
}

func SendProviderMsg() {
	providerSclient := sclientOpts[uint32(api.ChannelType_PROVIDER)].Sclient
	ctx := context.Background()
	msg := &sxapi.MbusMsg{
		ArgJson: "test",
	}
	log.Printf("Send Provider Info %d", msg.GetTargetId())
	providerSclient.SendMbusMsg(ctx, msg)
}

func SendSimAPIMsg() {
	sclient := sclientOpts[uint32(api.ChannelType_CLOCK)].Sclient
	simapi.GetAgentRequest(sclient, []uint64{})
}

/*func (bc *BaseCallback) SetAgentRequest() {
	fmt.Printf("before")
}*/

/*type CB2 struct {
	BaseCallback
}

func NewCBCallback() Callback {
	var bc2 Callback
	bc2 = &CB2{}
	return bc2
}*/

/*type Callback struct {
	util.AgentCallback
}

func NewAgentCallback(name string) util.AgentCallbackInterface {
	var bc2 util.AgentCallbackInterface
	bc2 = &Callback{
		util.NewAgentCallback(name),
	}
	return bc2
}

func (bc Callback) SetAgentRequest() {
	fmt.Printf("after %s\n", bc.Name)
}*/
/*type AgentCallback struct {
	util.AgentCallback
}

func NewAgentCallback(name string) util.AgentCallbackInterface {
	var bc2 util.AgentCallbackInterface
	bc2 = AgentCallback{
		util.NewAgentCallback(name),
	}
	return bc2
}

func (bc AgentCallback) SetAgentRequest() {
	fmt.Printf("after %s\n", bc.Name)
}

func runCallback() {
	ac2 := NewAgentCallback("rui")
	//ac2.SetAgentRequest()
	//ac2.Callback()
	bc := util.NewBaseCallback(ac2)
	bc.AgentCallback()
}*/

type AgentCallback struct {
	util.AgentCallback
}

func NewAgentCallback(simapi *api.SimAPI) util.AgentCallbackInterface {
	var agcb util.AgentCallbackInterface
	agcb = AgentCallback{
		util.NewAgentCallback(simapi),
	}
	return agcb
}

type ClockCallback struct {
	util.ClockCallback
}

func NewClockCallback(simapi *api.SimAPI) util.ClockCallbackInterface {
	var clcb util.ClockCallbackInterface
	clcb = ClockCallback{
		util.NewClockCallback(simapi),
	}
	return clcb
}

type ProviderCallback struct {
	util.ProviderCallback
}

func NewProviderCallback(simapi *api.SimAPI) util.ProviderCallbackInterface {
	var prcb util.ProviderCallbackInterface
	prcb = ProviderCallback{
		util.NewProviderCallback(simapi),
	}
	return prcb
}

type AreaCallback struct {
	util.AreaCallback
}

func NewAreaCallback(simapi *api.SimAPI) util.AreaCallbackInterface {
	var arcb util.AreaCallbackInterface
	arcb = AreaCallback{
		util.NewAreaCallback(simapi),
	}
	return arcb
}

/*type Callback struct {
	util.AreaCallback
	util.ProviderCallback
	util.ClockCallback
	util.AgentCallback
}

func NewCallback(simapi *api.SimAPI) util.Callback {
	var arcb util.CallbackInterface
	arcb = Callback{
		util.NewAreaCallback(simapi),
		util.NewAgentCallback(simapi),
		util.NewProviderCallback(simapi),
		util.NewClockCallback(simapi),
	}
	return arcb
}*/

func runCallback() {
	agcb := NewAgentCallback(simapi)
	clcb := NewClockCallback(simapi)
	prcb := NewProviderCallback(simapi)
	arcb := NewAreaCallback(simapi)
	//ac2.SetAgentRequest()
	//ac2.Callback()
	bc := util.NewBaseCallback(simapi, agcb, prcb, clcb, arcb)
	fmt.Printf("bc: ", bc)
	//bc.AgentCallback()
}

func main() {
	fmt.Printf("test\n")
	//runCallback()
	//cb2 := NewCBCallback()
	//cb2.AgentCallback()
	//cb2.SetAgentRequest()
	wg2 := sync.WaitGroup{} // for syncing other goroutines
	wg2.Add(1)
	wg2.Wait()
	// Register Node Server
	nodesrv := "127.0.0.1:9990"
	channelTypes := []uint32{}
	for _, opt := range sclientOpts {
		channelTypes = append(channelTypes, opt.ChType)
	}
	util.RegisterNodeLoop(nodesrv, "TestProvider", channelTypes)

	// Register Synerex Server
	sxServerAddress := "127.0.0.1:10000"
	client := util.RegisterSynerexLoop(sxServerAddress)
	util.RegisterSXServiceClients(client, sclientOpts)

	for {
		//SendClockMsg()
		//SendProviderMsg()
		SendSimAPIMsg()
		time.Sleep(2 * time.Second)
	}

	wg := sync.WaitGroup{} // for syncing other goroutines
	wg.Add(1)
	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!
}
