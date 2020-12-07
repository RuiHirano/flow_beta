package main

// main synerex serverからgatewayを介してother synerex serverへ情報を送る
// 基本的に一方通行

import (
	"flag"
	//"fmt"

	//"log"
	"os"

	//"strings"
	"runtime"
	"sync"
	"time"

	api "github.com/RuiHirano/flow_beta/api"
	util "github.com/RuiHirano/flow_beta/util"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	sxapi "github.com/synerex/synerex_api"
	sxutil "github.com/synerex/synerex_sxutil"
)

var (
	waiter          *api.Waiter
	mu              sync.Mutex
	logger          *util.Logger
	gatewayProvider  *GatewayProvider

	worker1Servaddr           = flag.String("servaddr", getServerAddress(), "The Synerex Server Listening Address")
	worker1Nodeaddr           = flag.String("nodeaddr", getNodeservAddress(), "Node ID Server Address")
	worker2Servaddr     = flag.String("workerServaddr", getWorkerServerAddress(), "Worker Synerex Server Listening Address")
	worker2Nodeaddr     = flag.String("workerNodeaddr", getWorkerNodeservAddress(), "Worker Node ID Server Address")
	providerName       = flag.String("providerName", getProviderName(), "Provider Name")
)

func getNodeservAddress() string {
	env := os.Getenv("SX_NODESERV_ADDRESS")
	if env != "" {
		return env
	} else {
		return "127.0.0.1:9990"
	}
}

func getServerAddress() string {
	env := os.Getenv("SX_SERVER_ADDRESS")
	if env != "" {
		return env
	} else {
		return "127.0.0.1:10000"
	}
}

func getWorkerNodeservAddress() string {
	env := os.Getenv("SX_WORKER_NODESERV_ADDRESS")
	if env != "" {
		return env
	} else {
		return "127.0.0.1:9990"
	}
}

func getWorkerServerAddress() string {
	env := os.Getenv("SX_WORKER_SERVER_ADDRESS")
	if env != "" {
		return env
	} else {
		return "127.0.0.1:10000"
	}
}

func getProviderName() string {
	env := os.Getenv("PROVIDER_NAME")
	if env != "" {
		return env
	} else {
		return "GatewayProvider"
	}
}

func init() {
	flag.Parse()

	//flag.Parse()
	logger = util.NewLogger()
	waiter = api.NewWaiter()

}

////////////////////////////////////////////////////////////
////////////     Gateway Provider           ////////////////
///////////////////////////////////////////////////////////
type GatewayProvider struct {
	Worker1API *api.ProviderAPI
	Agent1Provider *api.Provider
	Worker2API *api.ProviderAPI
	Agent2Provider *api.Provider
}

func NewGatewayProvider(worker1api *api.ProviderAPI, worker2api *api.ProviderAPI) *GatewayProvider {
	ap := &GatewayProvider{
		Worker1API: worker1api,
		Agent1Provider: nil,
		Worker2API: worker2api,
		Agent2Provider: nil,
	}
	return ap
}

func (ap *GatewayProvider) Connect() error {
	ap.Worker1API.ConnectServer()
	ap.Worker1API.RegisterProvider()
	ap.Worker2API.ConnectServer()
	ap.Worker2API.RegisterProvider()
	return nil
}

// 
func (ap *GatewayProvider) UpdateProviders(providers []*api.Provider, name string) error {
	if name == "WORKER1" {
		for _, p := range providers {
			if ap.Agent1Provider == nil && p.GetType() == api.Provider_AGENT {
				mu.Lock()
				ap.Agent1Provider = p
				mu.Unlock()
				logger.Success("Update Agent1 Id: %d\n", p.Id)
			}
		}
	}else if name == "WORKER2" {
		for _, p := range providers {
			if ap.Agent2Provider == nil && p.GetType() == api.Provider_AGENT {
				mu.Lock()
				ap.Agent2Provider = p
				mu.Unlock()
				logger.Success("Update Agent2 Id: %d\n", p.Id)
			}
		}
	}
	return nil
}

// 
func (ap *GatewayProvider) GetAgents(senderId uint64) []*api.Agent {
	agents := []*api.Agent{}
	t1 := time.Now()
	if ap.Agent1Provider.Id == senderId{
	//if name == "WORKER1" {
		// worker2のagent-providerから取得
		targets := []uint64{ap.Agent2Provider.Id}
		agents, _ = ap.Worker2API.GetAgents(targets)
	}else if ap.Agent2Provider.Id == senderId{
		// worker2のagent-providerから取得
		targets := []uint64{ap.Agent1Provider.Id}
		agents, _ = ap.Worker1API.GetAgents(targets)
	}
	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("Get Agent! Duration: %v ms, Wait: %d ms", duration, interval-duration)
	}
	
	return agents
}


////////////////////////////////////////////////////////////
////////////         Worker1 Callback       ////////////////
///////////////////////////////////////////////////////////

type Worker1Callback struct {
	*api.Callback
}

func (cb *Worker1Callback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	//logger.Debug("update providers request\n")
	providers := simMsg.GetUpdateProvidersRequest().GetProviders()
	gatewayProvider.UpdateProviders(providers, "WORKER1")
}

func (cb *Worker1Callback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*api.Agent {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	logger.Debug("gateway1: %v", simMsg)
	agents := gatewayProvider.GetAgents(simMsg.SenderId)
	return agents
}

///////////////////////////////////////////////////////////
////////////         Worker1 Callback       ////////////////
///////////////////////////////////////////////////////////

type Worker2Callback struct {
	*api.Callback
}

func (cb *Worker2Callback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	//logger.Debug("update providers request\n")
	providers := simMsg.GetUpdateProvidersRequest().GetProviders()
	gatewayProvider.UpdateProviders(providers, "WORKER2")
}

func (cb *Worker2Callback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*api.Agent {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	logger.Debug("gateway2: %v", simMsg)
	agents := gatewayProvider.GetAgents(simMsg.SenderId)
	return agents
}

func main() {
	logger.Info("Start Gateway Provider")
	logger.Info("NumCPU=%d", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())

	wg := sync.WaitGroup{} // for syncing other goroutines
	wg.Add(1)

	// Worker
	uid, _ := uuid.NewRandom()
	myProvider := &api.Provider{
		Id:   uint64(uid.ID()),
		Name: "GatewayProvider",
		Type: api.Provider_GATEWAY,
	}
	cb := api.NewCallback()

	// Worker Server Main
	wocb := &Worker1Callback{cb} // override
	worker1API := api.NewProviderAPI(myProvider, *worker1Servaddr, *worker1Nodeaddr, wocb)
	//worker1API.ConnectServer()
	//worker1API.RegisterProvider()

	// Worker Server Sub
	wocb2 := &Worker2Callback{cb} // override
	worker2API := api.NewProviderAPI(myProvider, *worker2Servaddr, *worker2Nodeaddr, wocb2)
	//worker2API.ConnectServer()
	//worker2API.RegisterProvider()

	// GatewayProvider
	gatewayProvider = NewGatewayProvider(worker1API, worker2API)
	gatewayProvider.Connect()

	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!
	logger.Success("Terminate Gateway Provider")

}
