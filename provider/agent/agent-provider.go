package main

import (
	//"context"

	"flag"
	"fmt"
	"log"

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
	agentsMessage  *Message
	myArea         *api.Area
	agentType      api.AgentType

	sclientOptsVis    map[uint32]*util.SclientOpt
	sclientOptsWorker map[uint32]*util.SclientOpt
	simapi            *api.SimAPI

	servaddr     = flag.String("servaddr", getServerAddress(), "The Synerex Server Listening Address")
	nodeaddr     = flag.String("nodeaddr", getNodeservAddress(), "Node ID Server Address")
	visServaddr  = flag.String("visServaddr", getVisServerAddress(), "Vis Synerex Server Listening Address")
	visNodeaddr  = flag.String("visNodeaddr", getVisNodeservAddress(), "Vis Node ID Server Address")
	providerName = flag.String("providerName", getProviderName(), "Provider Name")
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

func init() {
	flag.Parse()
	logger = util.NewLogger()
	agentsMessage = NewMessage()

	uid, _ := uuid.NewRandom()
	myProvider := &api.Provider{
		Id:   uint64(uid.ID()),
		Name: "AgentProvider",
		Type: api.Provider_AGENT,
	}
	simapi = api.NewSimAPI(myProvider)
	pm = util.NewProviderManager(myProvider)
	log.Printf("ProviderID: %d", simapi.Provider.Id)

	cb := util.NewCallback()
	vicb := &WorkerCallback{cb} // override
	sclientOptsVis = map[uint32]*util.SclientOpt{
		uint32(api.ChannelType_CLOCK): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_CLOCK),
			MBusCallback: util.GetClockCallback(simapi, vicb),
			ArgJson:      fmt.Sprintf("{Client:AgentProvider_Clock}"),
		},
		uint32(api.ChannelType_PROVIDER): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_PROVIDER),
			MBusCallback: util.GetProviderCallback(simapi, vicb),
			ArgJson:      fmt.Sprintf("{Client:AgentProvider_Provider}"),
		},
		uint32(api.ChannelType_AGENT): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AGENT),
			MBusCallback: util.GetAgentCallback(simapi, vicb),
			ArgJson:      fmt.Sprintf("{Client:AgentProvider_Agent}"),
		},
		uint32(api.ChannelType_AREA): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AREA),
			MBusCallback: util.GetAreaCallback(simapi, vicb),
			ArgJson:      fmt.Sprintf("{Client:AgentProvider_Area}"),
		},
	}

	wocb := &WorkerCallback{cb} // override
	sclientOptsWorker = map[uint32]*util.SclientOpt{
		uint32(api.ChannelType_CLOCK): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_CLOCK),
			MBusCallback: util.GetClockCallback(simapi, wocb),
			ArgJson:      fmt.Sprintf("{Client:AgentProvider_Clock}"),
		},
		uint32(api.ChannelType_PROVIDER): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_PROVIDER),
			MBusCallback: util.GetProviderCallback(simapi, wocb),
			ArgJson:      fmt.Sprintf("{Client:AgentProvider_Provider}"),
		},
		uint32(api.ChannelType_AGENT): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AGENT),
			MBusCallback: util.GetAgentCallback(simapi, wocb),
			ArgJson:      fmt.Sprintf("{Client:AgentProvider_Agent}"),
		},
		uint32(api.ChannelType_AREA): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AREA),
			MBusCallback: util.GetAreaCallback(simapi, wocb),
			ArgJson:      fmt.Sprintf("{Client:AgentProvider_Area}"),
		},
	}

	areaJson := os.Getenv("AREA")
	bytes := []byte(areaJson)
	json.Unmarshal(bytes, &myArea)
	//fmt.Printf("myArea: %v\n", myArea)

	agentType = api.AgentType_PEDESTRIAN
}

////////////////////////////////////////////////////////////
////////////            Message Class           ///////////
///////////////////////////////////////////////////////////

type Message struct {
	ready  chan struct{}
	agents []*api.Agent
}

func NewMessage() *Message {
	return &Message{ready: make(chan struct{}), agents: make([]*api.Agent, 0)}
}

func (m *Message) Set(a []*api.Agent) {
	m.agents = a
	close(m.ready)
}

func (m *Message) Get() []*api.Agent {
	select {
	case <-m.ready:
		//case <-time.After(100 * time.Millisecond):
		//	logger.Warn("Timeout Get")
	}

	return m.agents
}

func forwardClock() {
	//senderId := myProvider.Id
	t1 := time.Now()
	//logger.Debug("1: 同エリアエージェント取得")
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	filters := []*api.Filter{}
	for _, target := range targets {
		filters = append(filters, &api.Filter{TargetId: target})
	}
	sclient := sclientOptsWorker[uint32(api.ChannelType_AGENT)].Sclient
	sameAgents := []*api.Agent{}
	//if len(targets) != 0 {
	simMsgs, _ := simapi.GetAgentRequest(sclient, filters)
	////logger.Debug("1: targets %v\n", targets)
	for _, simMsg := range simMsgs {
		agents := simMsg.GetGetAgentResponse().GetAgents()
		sameAgents = append(sameAgents, agents...)
	}
	//}

	// [2. Calculation]次の時間のエージェントを計算する
	//logger.Debug("2: エージェント計算を行う")
	nextControlAgents := sim.ForwardStep(sameAgents) // agents in control area
	//logger.Debug("2: Set")
	agentsMessage.Set(nextControlAgents)

	// databaseに保存
	/*targets = pm.GetProviderIds([]api.Provider_Type{
		api.Provider_DATABASE,
	})
	simapi.SetAgentRequest(myProvider.Id, targets, nextControlAgents)*/

	// visに保存
	targets = pm.GetProviderIds([]api.Provider_Type{
		api.Provider_VISUALIZATION,
	})
	filters = []*api.Filter{}
	for _, target := range targets {
		filters = append(filters, &api.Filter{TargetId: target})
	}
	sclient = sclientOptsVis[uint32(api.ChannelType_AGENT)].Sclient
	simapi.SetAgentRequest(sclient, filters, nextControlAgents)

	//logger.Debug("3: 隣接エージェントを取得")
	targets = pm.GetProviderIds([]api.Provider_Type{
		api.Provider_GATEWAY,
	})
	filters = []*api.Filter{}
	for _, target := range targets {
		filters = append(filters, &api.Filter{TargetId: target})
	}
	sclient = sclientOptsWorker[uint32(api.ChannelType_AGENT)].Sclient

	neighborAgents := []*api.Agent{}
	//if len(targets) != 0 {
	simMsgs, _ = simapi.GetAgentRequest(sclient, filters)

	for _, simMsg := range simMsgs {
		agents := simMsg.GetGetAgentResponse().GetAgents()
		neighborAgents = append(neighborAgents, agents...)
	}
	//}

	// [4. Update Agents]重複エリアのエージェントを更新する
	nextAgents := sim.UpdateDuplicateAgents(nextControlAgents, neighborAgents)
	// Agentsをセットする
	sim.SetAgents(nextAgents)

	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("Forward Clock! Duration: %v ms, Wait: %d ms", duration, interval-duration)
	}
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
	//logger.Debug("Got Forward Clock Init Request")
	// 同じエリアからエージェント情報を取得
	agentsMessage = NewMessage()
	logger.Success("Forward Clock Init")
}

func (cb *WorkerCallback) ForwardClockMainRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	//logger.Debug("Got Forward Clock Main Request")
	// エージェント情報を計算する
	forwardClock()
	logger.Success("Forward Clock Main")
}

func (cb *WorkerCallback) ForwardClockTerminateRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
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
	//logger.Debug("Got Set Agent Request")
	// Agentをセットする
	agents := simMsg.GetSetAgentRequest().GetAgents()

	// Agent情報を追加する
	sim.AddAgents(agents)
	logger.Success("Set Agent %d", len(agents))
}

func (cb *WorkerCallback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*api.Agent {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	//logger.Debug("Got Get Agent Request")

	agents := sim.Agents
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
	//logger.Debug("Got Get Agent Request")

	agents := sim.Agents
	return agents
}

func main() {
	logger.Info("Start Agent Provider")
	logger.Info("NumCPU=%d", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Connect to Worker Syenrex Node Server
	// Register Node Server
	channelTypes := []uint32{}
	for _, opt := range sclientOptsWorker {
		channelTypes = append(channelTypes, opt.ChType)
	}
	ni := sxutil.GetDefaultNodeServInfo()
	util.RegisterNodeLoop(ni, *nodeaddr, "AgentProvider", channelTypes)

	// Register Synerex Server
	client := util.RegisterSynerexLoop(*servaddr)
	util.RegisterSXServiceClients(ni, client, sclientOptsWorker)
	logger.Success("Subscribe Mbus")

	// Connect to Master Syenrex Node Server
	// Register Node Server

	channelTypes = []uint32{}
	for _, opt := range sclientOptsVis {
		channelTypes = append(channelTypes, opt.ChType)
	}
	ni = sxutil.NewNodeServInfo()
	util.RegisterNodeLoop(ni, *visNodeaddr, "WorkerProvider", channelTypes)

	// Register Synerex Server
	client = util.RegisterSynerexLoop(*visServaddr)
	util.RegisterSXServiceClients(ni, client, sclientOptsVis)
	logger.Success("Subscribe Mbus")

	wg := sync.WaitGroup{} // for syncing other goroutines
	wg.Add(1)

	// Simulator
	sim = NewSimulator(myArea, api.AgentType_PEDESTRIAN)

	time.Sleep(5 * time.Second)

	sclient := sclientOptsWorker[uint32(api.ChannelType_PROVIDER)].Sclient
	workerProvider = util.RegisterProviderLoop(sclient, simapi)
	logger.Success("Register Provider to Worker Provider at %d", workerProvider.Id)

	sclient = sclientOptsVis[uint32(api.ChannelType_PROVIDER)].Sclient
	visProvider = util.RegisterProviderLoop(sclient, simapi)
	logger.Success("Register Provider to Vis Provider at %d", visProvider.Id)

	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!
	logger.Success("Terminate Agent Provider")
}
