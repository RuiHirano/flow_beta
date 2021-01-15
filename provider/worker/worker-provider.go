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
	pm             *api.ProviderManager
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
	MasterAPI *api.ProviderAPI
	WorkerAPI *api.ProviderAPI
}

func NewWorkerProvider(masterapi *api.ProviderAPI, workerapi *api.ProviderAPI) *WorkerProvider {
	ap := &WorkerProvider{
		MasterAPI: masterapi,
		WorkerAPI: workerapi,
	}
	return ap
}
func (ap *WorkerProvider) Connect(){
	if err := ap.WorkerAPI.ConnectServer(false); err != nil{
		logger.Error("error on Connect: WorkerServer", err)
	}
	logger.Success("Connect to WorkerServer")
	if err := ap.MasterAPI.ConnectServer(true); err != nil{
		logger.Error("error on Connect MasterServer: ", err)
	}
	logger.Success("Connect to MasterServer")
	if err := ap.MasterAPI.RegisterProvider(); err != nil{
		logger.Error("error on Connect RegisterProvider: ", err)
	}
	logger.Success("Connect to RegisterProvider")
		logger.Success("Success Connect: ")

}

// 
func (ap *WorkerProvider) RegisterProvider(provider *api.Provider) {
	//logger.Debug("calcNextAgents 0")
	pm.AddProvider(provider)
	//fmt.Printf("regist provider! %v %v\n", p.GetId(), p.GetType())

	// update provider to worker
	targets := pm.GetTargets([]api.Provider_Type{
		api.Provider_GATEWAY,
		api.Provider_AGENT,
	})
	providers := pm.GetProviders()
	if err := ap.WorkerAPI.UpdateProviders(targets, providers); err != nil{
		logger.Error("error on Connect RegisterProvider: ", err)
	}else{
		logger.Success("Success RegisterProvider: ")
	}
}

// 
func (ap *WorkerProvider) SimulatorRequest(simReq *api.SimulatorRequest) {
	targets := pm.GetTargets([]api.Provider_Type{
		api.Provider_AGENT,
	})
	if err := ap.WorkerAPI.SimulatorRequest(targets, simReq); err != nil{
		logger.Error("error on Connect SimulatorRequest: ", err)
	}else{
		logger.Success("Success SimulatorRequest: ")
	}
}

// 
func (ap *WorkerProvider) Reset() {
	targets := pm.GetTargets([]api.Provider_Type{
		api.Provider_AGENT,
	})
	if err := ap.WorkerAPI.Reset(targets); err != nil{
		logger.Error("error on Connect Reset: ", err)
	}else{
		logger.Success("Success Reset: ")
	}
}

// 
func (ap *WorkerProvider) SetClock(clock *api.Clock) {
	targets := pm.GetTargets([]api.Provider_Type{
		api.Provider_AGENT,
	})
	if err := ap.WorkerAPI.SetClock(targets, clock); err != nil{
		logger.Error("error on SetClock: ", err)
	}else{
		logger.Success("Success Set Clock: ")
	}
}

// 
func (ap *WorkerProvider) SetAgents(agents []*api.Agent) {
	//agents := make([]*api.Agent, 0)

	targets := pm.GetTargets([]api.Provider_Type{
		api.Provider_AGENT,
	})
	if err := ap.WorkerAPI.SetAgents(targets, agents); err != nil{
		logger.Error("error on Set Agents: ", err)
	}else{
		logger.Success("Success Set Agents: ")
	}
}

// 
func (ap *WorkerProvider) ForwardClockInit(){
	
	targets := pm.GetTargets([]api.Provider_Type{
		api.Provider_AGENT,
	})
	if err := ap.WorkerAPI.ForwardClockInit(targets); err != nil{
		logger.Error("error on Clock Init: ", err)
	}else{
		logger.Success("Success Forward Clock Init: ")
	}
}

// 
func (ap *WorkerProvider) ForwardClockMain() {
	targets := pm.GetTargets([]api.Provider_Type{
		api.Provider_AGENT,
	})
	if err := ap.WorkerAPI.ForwardClockMain(targets); err != nil{
		logger.Error("error on Clock Main: ", err)
	}else{
		logger.Success("Success Forward Clock Main: ")
	}
}

// 
func (ap *WorkerProvider) ForwardClockTerminate() {
	targets := pm.GetTargets([]api.Provider_Type{
		api.Provider_AGENT,
	})
	if err := ap.WorkerAPI.ForwardClockTerminate(targets); err != nil{
		logger.Error("error on ClockTerminate: ", err)
	}else{
		logger.Success("Success Forward Clock Terminate: ")
	}
}


////////////////////////////////////////////////////////////
////////////         Master Callback       ////////////////
///////////////////////////////////////////////////////////
type MasterCallback struct {
	*api.Callback
}

func (cb *MasterCallback) SimulatorRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	simReq := simMsg.GetSimulatorRequest()
	workerProvider.SimulatorRequest(simReq)
	logger.Success("Simulator\n")
}

func (cb *MasterCallback) ResetRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	workerProvider.Reset()
	logger.Success("Reset\n")
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
	*api.Callback
}

func (cb *WorkerCallback) RegisterProviderRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) *api.Provider {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	provider := simMsg.GetRegisterProviderRequest().GetProvider()
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
	pm = api.NewProviderManager(myProvider)
	cb := api.NewCallback()

	// Worker Server
	wocb := &WorkerCallback{cb} // override
	workerAPI := api.NewProviderAPI(myProvider, *servaddr, *nodeaddr, wocb)
	//workerAPI.ConnectServer()
	//workerAPI.RegisterProvider()

	// Master Server
	macb := &MasterCallback{cb} // override
	masterAPI := api.NewProviderAPI(myProvider, *masterServaddr, *masterNodeaddr, macb)
	//masterAPI.ConnectServer()
	//masterAPI.RegisterProvider()

	// WorkerProvider
	workerProvider = NewWorkerProvider(masterAPI, workerAPI)
	workerProvider.Connect()

	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!
	logger.Success("Terminate Worker Provider")

}
