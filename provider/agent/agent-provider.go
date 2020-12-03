package main

import (
	//"context"

	"flag"
	//"fmt"
	//"log"

	//"math/rand"
	"os"
	"sync"

	"time"

	//"runtime"
	"encoding/json"
	"runtime"

	api "github.com/RuiHirano/flow_beta/api"
	util "github.com/RuiHirano/flow_beta/util"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	sxapi "github.com/synerex/synerex_api"
	sxutil "github.com/synerex/synerex_sxutil"
)

var (
	myProvider     *api.Provider
	workerProvider *api.Provider
	visProvider    *api.Provider
	pm             *util.ProviderManager
	sim            *Simulator
	logger         *util.Logger
	mu             sync.Mutex
	myArea         *api.Area
	agentType      api.AgentType
	agentProvider  *AgentProvider

	simapi            *api.SimAPI

	servaddr     = flag.String("servaddr", getServerAddress(), "The Synerex Server Listening Address")
	nodeaddr     = flag.String("nodeaddr", getNodeservAddress(), "Node ID Server Address")
	visServaddr  = flag.String("visServaddr", getVisServerAddress(), "Vis Synerex Server Listening Address")
	visNodeaddr  = flag.String("visNodeaddr", getVisNodeservAddress(), "Vis Node ID Server Address")
	providerName = flag.String("providerName", getProviderName(), "Provider Name")
	areaJson     = flag.String("areaJson", getAreaJson(), "Area Information")
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

func getVisNodeservAddress() string {
	env := os.Getenv("SX_VIS_NODESERV_ADDRESS")
	if env != "" {
		return env
	} else {
		return "127.0.0.1:9990"
	}
}

func getVisServerAddress() string {
	env := os.Getenv("SX_VIS_SERVER_ADDRESS")
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
		return "AgentProvider"
	}
}

func getAreaJson() string {
	env := os.Getenv("AREA_JSON")
	if env != "" {
		return env
	} else {
		return ""
	}
}

func init() {
	flag.Parse()
	logger = util.NewLogger()


	//areaJson := os.Getenv("AREA")
	bytes := []byte(*areaJson)
	json.Unmarshal(bytes, &myArea)
	//fmt.Printf("myArea: %v\n", myArea)

	agentType = api.AgentType_PEDESTRIAN
}

////////////////////////////////////////////////////////////
////////////     Agent Provider        ////////////////
///////////////////////////////////////////////////////////
type AgentProvider struct {
	Simulator *Simulator
	API *util.WorkerAPI
}

func NewAgentProvider(simulator *Simulator, api *util.WorkerAPI) *AgentProvider {
	ap := &AgentProvider{
		Simulator: simulator,
		API: api,
	}
	return ap
}

// Connect: Worker Nodeに接続する
func (ap *AgentProvider) ForwardClock() error {
	//logger.Debug("calcNextAgents 0")
	t1 := time.Now()
	ap.Simulator.ForwardStep() // agents in control area
	//agentsMessage.Set(nextControlAgents)

	// visに保存
	/*targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_VISUALIZATION,
	})
	filters := []*api.Filter{}
	for _, target := range targets {
		filters = append(filters, &api.Filter{TargetId: target})
	}
	logger.Debug("SetAgentRequest %v", filters)
	sclient := sclientOptsVis[uint32(api.ChannelType_AGENT)].Sclient
	simapi.SetAgentRequest(sclient, filters, nextControlAgents)*/

	//logger.Debug("calcNextAgents 2")
	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("CalcNextAgents! Duration: %v ms, Wait: %d ms", duration, interval-duration)
	}
	return nil
}

func (ap *AgentProvider) GetSameAreaAgents() []*api.Agent {
	//logger.Debug("getSameAreaAgents 0")
	t1 := time.Now()
	//logger.Debug("1: 同エリアエージェント取得")
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	sameAgents := ap.API.GetAgents(targets)
	sim.SetDiffAgents(sameAgents)
	
	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("GetSameAreaAgents! Duration: %v ms, Wait: %d ms", duration, interval-duration)
	}
	return sameAgents
}


func (ap *AgentProvider) GetNeighborAreaAgents() []*api.Agent {
	//logger.Debug("getSameAreaAgents 0")
	t1 := time.Now()
	//logger.Debug("1: 同エリアエージェント取得")
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_GATEWAY,
	})
	neighborAgents := ap.API.GetAgents(targets)
	sim.SetDiffAgents(neighborAgents)
	
	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("GetSameAreaAgents! Duration: %v ms, Wait: %d ms", duration, interval-duration)
	}
	return neighborAgents
}

func (ap *AgentProvider) UpdateAgents(agents []*api.Agent) error {
	t1 := time.Now()
	
	newAgents := sim.UpdateDuplicateAgents(agents)
	// Agentsをセットする
	sim.SetAgents(newAgents)

	//logger.Debug("updateNextAgents 3")
	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("UpdateNextAgents! Duration: %v ms, Wait: %d ms", duration, interval-duration)
	}
	return nil
}

func (ap *AgentProvider) GetAgents() []*api.Agent {
	return ap.Simulator.Agents
}

func (ap *AgentProvider) AddAgents(agents []*api.Agent) int {
	ap.Simulator.AddAgents(agents)
	// FIX
	return len(agents)
}

////////////////////////////////////////////////////////////
////////////         Worker Callback       ////////////////
///////////////////////////////////////////////////////////
type WorkerCallback struct {
	*util.Callback
}

func (cb *WorkerCallback) ForwardClockInitRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	agentProvider.GetSameAreaAgents()
	logger.Success("Forward Clock Init")
}

func (cb *WorkerCallback) ForwardClockMainRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	agentProvider.ForwardClock()
	logger.Success("Forward Clock Main Agents: %d", len(sim.Agents))
}

func (cb *WorkerCallback) ForwardClockTerminateRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	agents := agentProvider.GetNeighborAreaAgents()
	agentProvider.UpdateAgents(agents)
	logger.Success("Forward Clock Terminate")
}

func (cb *WorkerCallback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	providers := simMsg.GetUpdateProvidersRequest().GetProviders()
	pm.SetProviders(providers)
	logger.Success("Update Workers num: %d\n", len(providers))
}

func (cb *WorkerCallback) SetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	// Agentをセットする
	agents := simMsg.GetSetAgentRequest().GetAgents()

	// Agent情報を追加する
	num := agentProvider.AddAgents(agents)
	logger.Success("Set Agent %d", num)
}

func (cb *WorkerCallback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*api.Agent {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	agents := agentProvider.GetAgents()
	logger.Success("Send %d Agent to %d", len(agents), simMsg.SenderId)
	return agents
}

////////////////////////////////////////////////////////////
////////////          Vis Callback         ////////////////
///////////////////////////////////////////////////////////
type VisCallback struct {
	*util.Callback
}

func (cb *VisCallback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*api.Agent {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	agents := sim.Agents
	return agents
}

func main() {
	logger.Info("Start Agent Provider")
	logger.Info("NumCPU=%d", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())

	wg := sync.WaitGroup{} // for syncing other goroutines
	wg.Add(1)

	// Simulator
	sim = NewSimulator(myArea, api.AgentType_PEDESTRIAN)

	// Worker Server
	uid, _ := uuid.NewRandom()
	myProvider := &api.Provider{
		Id:   uint64(uid.ID()),
		Name: "AgentProvider",
		Type: api.Provider_AGENT,
	}
	simapi = api.NewSimAPI(myProvider)
	cb := util.NewCallback()

	// Worker Server
	wocb := &WorkerCallback{cb} // override
	workerAPI := util.NewWorkerAPI(simapi, *servaddr, *nodeaddr, wocb)
	workerAPI.ConnectServer()
	workerAPI.RegisterProvider()

	// Vis Server
	vicb := &VisCallback{cb} // override
	visAPI := util.NewWorkerAPI(simapi, *servaddr, *nodeaddr, vicb)
	visAPI.ConnectServer()
	visAPI.RegisterProvider()

	// AgentProvider
	agentProvider = NewAgentProvider(sim, workerAPI)

	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!
	logger.Success("Terminate Agent Provider")
}
