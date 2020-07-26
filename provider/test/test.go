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

func main() {
	fmt.Printf("test")

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
