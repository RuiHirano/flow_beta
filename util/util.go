package util

import (
	"context"
	"log"
	"time"

	api "github.com/RuiHirano/flow_beta/api"
	sxapi "github.com/synerex/synerex_api"
	sxutil "github.com/synerex/synerex_sxutil"
)

type SclientOpt struct {
	Sclient      *sxutil.SXServiceClient
	ChType       uint32
	MBusCallback func(*sxutil.SXServiceClient, *sxapi.MbusMsg)
	ArgJson      string
	Providers    []*api.Provider
}

func RegisterNode(nodesrv string, chTypes []uint32) (string, error) {
	sxServerAddress, err := sxutil.RegisterNode(nodesrv, "TestProvoider", chTypes, nil)
	if err != nil {
		// error occour
		return "", err
	}
	log.Printf("Connecting SynerexServer at [%s]\n", sxServerAddress)

	go sxutil.HandleSigInt()
	sxutil.RegisterDeferFunction(sxutil.UnRegisterNode)

	return sxServerAddress, nil
}

// NodeServに繋がるまで繰り返す
func RegisterNodeLoop(nodesrv string, name string, chTypes []uint32) *sxutil.NodeServInfo {
	go sxutil.HandleSigInt() // Ctl+cを認識させる
	for {
		sxServerAddress, err := sxutil.RegisterNodeWithCmd(nodesrv, name, chTypes, nil, nil)
		if err != nil {
			log.Printf("Can't register node. reconeccting...\n")
			time.Sleep(1 * time.Second)
		} else {
			sxutil.RegisterDeferFunction(sxutil.UnRegisterNode)
			log.Printf("Connecting NodeServer at [%s]\n", sxServerAddress)
			ni := sxutil.GetDefaultNodeServInfo()
			return ni
		}
	}
}

func RegisterSXServiceClients(client sxapi.SynerexClient, opts map[uint32]*SclientOpt) map[uint32]*SclientOpt {
	for key, opt := range opts {
		sclient := sxutil.NewSXServiceClient(client, opt.ChType, opt.ArgJson) // service client
		sclient.MbusID = sxutil.IDType(opt.ChType)                            // MbusIDをChTypeに変更
		log.Printf("debug MbusID: %d", sclient.MbusID)
		opts[key].Sclient = sclient
		go SubscribeMbusLoop(sclient, opt.MBusCallback)
	}
	return opts
}

func SubscribeMbusLoop(sclient *sxutil.SXServiceClient, mbcb func(*sxutil.SXServiceClient, *sxapi.MbusMsg)) {
	//called as goroutine
	ctx := context.Background() // should check proper context
	sxutil.RegisterDeferFunction(func() {
		log.Println("Mbus Closing...")
		sclient.CloseMbus(ctx)
	})
	for {
		sclient.SubscribeMbus(ctx, mbcb)
		// comes here if channel closed
		log.Println("SMarket Server Closed? Reconnecting...")
		time.Sleep(1 * time.Second)
	}
}

// Synerexに繋がるまで繰り返す
func RegisterSynerexLoop(sxServerAddress string) sxapi.SynerexClient {
	for {
		client := sxutil.GrpcConnectServer(sxServerAddress)
		if client == nil {
			log.Printf("Can't register synerex. reconeccting...\n")
			time.Sleep(1 * time.Second)
		} else {
			log.Printf("Register to %s\n", sxServerAddress)
			return client
		}
	}
}
