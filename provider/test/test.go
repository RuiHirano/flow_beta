package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	napi "github.com/RuiHirano/synerex_simulation_beta/api"
	api "github.com/synerex/synerex_api"
	proto "github.com/synerex/synerex_proto"
	sxutil "github.com/synerex/synerex_sxutil"
)

func supplyCallback(clt *sxutil.SXServiceClient, sp *api.Supply) {
	log.Println("Got RPA User supply callback")
}

func subscribeSupply(client *sxutil.SXServiceClient) {
	//called as goroutine
	ctx := context.Background() // should check proper context
	client.SubscribeSupply(ctx, supplyCallback)
	// comes here if channel closed
	log.Println("SMarket Server Closed?")
}

func main() {
	fmt.Printf("test")

	// connect NodeServ
	nodesrv := "127.0.0.1:9990"
	nodeapi := napi.NewNodeAPI(nil)
	nodeapi.ConnectServer(nodesrv)
	go sxutil.HandleSigInt()
	sxutil.RegisterDeferFunction(sxutil.UnRegisterNode)

	channelTypes := []uint32{proto.MEETING_SERVICE}
	// obtain synerex server address from nodeserv

	srv, err := sxutil.RegisterNode(nodesrv, "RPAUserProvider", channelTypes, nil)
	if err != nil {
		log.Fatal("Can't register node...")
	}
	log.Printf("Connecting Server [%s]\n", srv)

	wg := sync.WaitGroup{} // for syncing other goroutines
	//sxServerAddress = srv
	client := sxutil.GrpcConnectServer(srv)
	argJson := fmt.Sprintf("{Client:RPAUser}")
	sclient := sxutil.NewSXServiceClient(client, proto.MEETING_SERVICE, argJson)

	wg.Add(1)
	go subscribeSupply(sclient)

	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!
}
