package main

import (
	"flag"

	//"math/rand"
	"os"
	"sync"

	"runtime"
	"time"

	api "github.com/RuiHirano/flow_beta/api"
	util "github.com/RuiHirano/flow_beta/util"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	sxapi "github.com/synerex/synerex_api"
	sxutil "github.com/synerex/synerex_sxutil"
)

var (
	myProvider     *api.Provider
	mu             sync.Mutex
	logger         *util.Logger
	pm             *util.ProviderManager
	workerProvider  *WorkerProvider

	servaddr          = flag.String("servaddr", getServerAddress(), "The Synerex Server Listening Address")
	nodeaddr          = flag.String("nodeaddr", getNodeservAddress(), "Node ID Server Address")
	masterServaddr    = flag.String("masterServaddr", getMasterServerAddress(), "Master Synerex Server Listening Address")
	masterNodeaddr    = flag.String("masterNodeaddr", getMasterNodeservAddress(), "Master Node ID Server Address")
	providerName      = flag.String("providerName", getProviderName(), "Provider Name")
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

func getMasterNodeservAddress() string {
	env := os.Getenv("SX_MASTER_NODESERV_ADDRESS")
	if env != "" {
		return env
	} else {
		return "127.0.0.1:9990"
	}
}

func getMasterServerAddress() string {
	env := os.Getenv("SX_MASTER_SERVER_ADDRESS")
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
		return "WorkerProvider"
	}
}

const MAX_AGENTS_NUM = 1000

func init() {
	flag.Parse()


	logger = util.NewLogger()
	logger.SetPrefix("Scenario")

}

////////////////////////////////////////////////////////////
////////////     Worker Provider           ////////////////
///////////////////////////////////////////////////////////
type WorkerProvider struct {
	MasterAPI *util.MasterAPI
	WorkerAPI *util.WorkerAPI
}

func NewWorkerProvider(masterapi *util.MasterAPI, workerapi *util.WorkerAPI) *WorkerProvider {
	ap := &WorkerProvider{
		MasterAPI: masterapi,
		WorkerAPI: workerapi,
	}
	return ap
}

// 
func (ap *WorkerProvider) RegisterProvider(provider *api.Provider) error {
	//logger.Debug("calcNextAgents 0")
	pm.AddProvider(provider)
	//fmt.Printf("regist provider! %v %v\n", p.GetId(), p.GetType())

	// update provider to worker
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_GATEWAY,
		api.Provider_AGENT,
	})
	providers := pm.GetProviders()
	ap.WorkerAPI.UpdateProviders(targets, providers)
	logger.Success("Update Providers! Worker Num: ", len(targets))
	return nil
}

// 
func (ap *WorkerProvider) SetClock(clock *api.Clock) error {
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	ap.WorkerAPI.SetClock(targets, clock)
	logger.Success("Set Clock at %d", clock.GlobalTime)
	return nil
}

// 
func (ap *WorkerProvider) SetAgents(agents []*api.Agent) error {
	//agents := make([]*api.Agent, 0)

	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	ap.WorkerAPI.SetAgents(targets, agents)
	return nil
}

// 
func (ap *WorkerProvider) ForwardClockInit() error {
	
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	ap.WorkerAPI.ForwardClockInit(targets)
	return nil
}

// 
func (ap *WorkerProvider) ForwardClockMain() error {
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	ap.WorkerAPI.ForwardClockMain(targets)
	return nil
}

// 
func (ap *WorkerProvider) ForwardClockTerminate() error {
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	ap.WorkerAPI.ForwardClockTerminate(targets)
	return nil
}


////////////////////////////////////////////////////////////
////////////         Master Callback       ////////////////
///////////////////////////////////////////////////////////
type MasterCallback struct {
	*util.Callback
}

func (cb *MasterCallback) ForwardClockInitRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	t1 := time.Now()
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	workerProvider.ForwardClockInit()
	
	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("Forward Clock Init! Duration: %v ms, Wait: %d ms", duration, interval-duration)
	}
}

func (cb *MasterCallback) ForwardClockMainRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	t1 := time.Now()
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	workerProvider.ForwardClockMain()
	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("Forward Clock Main! Duration: %v ms, Wait: %d ms", duration, interval-duration)
	}
}

func (cb *MasterCallback) ForwardClockTerminateRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	t1 := time.Now()
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	workerProvider.ForwardClockTerminate()
	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("Forward Clock Terminate! Duration: %v ms, Wait: %d ms", duration, interval-duration)
	}
}

func (cb *MasterCallback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	providers := simMsg.GetUpdateProvidersRequest().GetProviders()
	logger.Success("Update Workers num: %d\n", len(providers))
}

func (cb *MasterCallback) SetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)

	// request to providers
	agents := simMsg.GetSetAgentRequest().GetAgents()
	workerProvider.SetAgents(agents)

	logger.Success("Set Agents Add: %v", len(agents))
}

func (cb *MasterCallback) SetClockRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)

	// request to providers
	clock := simMsg.GetSetClockRequest().GetClock()
	workerProvider.SetClock(clock)

	logger.Success("Set Clock at %d", clock.GlobalTime)
}

////////////////////////////////////////////////////////////
////////////         Worker Callback       ////////////////
///////////////////////////////////////////////////////////
type WorkerCallback struct {
	*util.Callback
}

func (cb *WorkerCallback) RegisterProviderRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) *api.Provider {
	simMsg := &api.SimMsg{}
	provider := simMsg.GetRegisterProviderRequest().GetProvider()
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	workerProvider.RegisterProvider(provider)

	return workerProvider.WorkerAPI.SimAPI.Provider
}

func main() {
	logger.Info("Start Worker Provider")
	logger.Info("NumCPU=%d", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())

	wg := sync.WaitGroup{} // for syncing other goroutines
	wg.Add(1)

	// Worker
	uid, _ := uuid.NewRandom()
	myProvider := &api.Provider{
		Id:   uint64(uid.ID()),
		Name: "WorkerProvider",
		Type: api.Provider_WORKER,
	}
	pm = util.NewProviderManager(myProvider)
	simapi := api.NewSimAPI(myProvider)
	cb := util.NewCallback()

	// Worker Server
	wocb := &WorkerCallback{cb} // override
	workerAPI := util.NewWorkerAPI(simapi, *servaddr, *nodeaddr, wocb)
	workerAPI.ConnectServer()
	workerAPI.RegisterProvider()

	// Master Server
	macb := &MasterCallback{cb} // override
	masterAPI := util.NewMasterAPI(simapi, *masterServaddr, *masterNodeaddr, macb)
	masterAPI.ConnectServer()
	masterAPI.RegisterProvider()

	// WorkerProvider
	workerProvider = NewWorkerProvider(masterAPI, workerAPI)

	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!
	logger.Success("Terminate Worker Provider")

}
