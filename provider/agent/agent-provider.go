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
		Name: "MasterServer",
		Type: api.Provider_MASTER,
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
	fmt.Printf("myArea: %v\n", myArea)

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

////////////////////////////////////////////////////////////
////////////            Message Class2           ///////////
///////////////////////////////////////////////////////////

/*type Message2 struct {
	isFinish bool
	agents   []*api.Agent
}

func NewMessage2() *Message2 {
	return &Message2{isFinish: false, agents: make([]*api.Agent, 0)}
}

func (m *Message2) Set(a []*api.Agent) {
	m.agents = a
	m.isFinish = true
}

func (m *Message2) Get() []*api.Agent {
	for {
		if m.isFinish == true {
			time.Sleep(1 * time.Millisecond)
			break
		}
	}

	return m.agents
}*/

func forwardClock() {
	//senderId := myProvider.Id
	//t1 := time.Now()
	//logger.Debug("1: 同エリアエージェント取得")
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	sclient := sclientOptsWorker[uint32(api.ChannelType_AGENT)].Sclient
	sameAgents := []*api.Agent{}
	if len(targets) != 0 {
		simMsgs, _ := simapi.GetAgentRequest(sclient, targets)
		////logger.Debug("1: targets %v\n", targets)
		for _, simMsg := range simMsgs {
			agents := simMsg.GetGetAgentResponse().GetAgents()
			sameAgents = append(sameAgents, agents...)
		}
	}

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
	sclient = sclientOptsVis[uint32(api.ChannelType_AGENT)].Sclient
	simapi.SetAgentRequest(sclient, targets, nextControlAgents)

	//logger.Debug("3: 隣接エージェントを取得")
	targets = pm.GetProviderIds([]api.Provider_Type{
		api.Provider_GATEWAY,
	})
	sclient = sclientOptsWorker[uint32(api.ChannelType_AGENT)].Sclient

	neighborAgents := []*api.Agent{}
	if len(targets) != 0 {
		simMsgs, _ := simapi.GetAgentRequest(sclient, targets)
		////logger.Debug("3: targets %v\n", targets)
		for _, simMsg := range simMsgs {
			agents := simMsg.GetGetAgentResponse().GetAgents()
			neighborAgents = append(neighborAgents, agents...)
		}
	}

	//logger.Debug("4: エージェントを更新")
	// [4. Update Agents]重複エリアのエージェントを更新する
	nextAgents := sim.UpdateDuplicateAgents(nextControlAgents, neighborAgents)
	// Agentsをセットする
	sim.SetAgents(nextAgents)

	//logger.Info("Finish: Clock Forwarded. AgentNum:  %v", len(nextControlAgents))
	logger.Info("\x1b[32m\x1b[40m [ Agent : %v ] \x1b[0m", len(nextControlAgents))
	//t2 := time.Now()
	//duration := t2.Sub(t1).Milliseconds()
	//logger.Info("Duration: %v, PID: %v", duration, myProvider.Id)
}

// callback for each Supply
/*func demandCallback(clt *api.SMServiceClient, dm *api.Demand) {
	switch dm.GetSimDemand().GetType() {

	case api.MsgType_UPDATE_PROVIDERS_REQUEST:
		providers := dm.GetSimDemand().GetUpdateProvidersRequest().GetProviders()
		pm.SetProviders(providers)

		// response
		targets := []uint64{dm.GetSimDemand().GetSenderId()}
		senderId := myProvider.Id
		msgId := dm.GetSimDemand().GetMsgId()
		simapi.UpdateProvidersResponse(senderId, targets, msgId)
		//logger.Info("Finish: Update Providers num: %v\n", len(providers))
		//for _, p := range providers {
		//logger.Debug("PID: %v,  Name: %v\n", p.Id, p.Name)
		//}

	case api.MsgType_SET_AGENT_REQUEST:
		// Agentをセットする
		agents := dm.GetSimDemand().GetSetAgentRequest().GetAgents()

		// Agent情報を追加する
		sim.AddAgents(agents)

		// セット完了通知を送る
		targets := []uint64{dm.GetSimDemand().GetSenderId()}
		senderId := myProvider.Id
		msgId := dm.GetSimDemand().GetMsgId()
		simapi.SetAgentResponse(senderId, targets, msgId)
		//logger.Info("\x1b[32m\x1b[40m [ Agent : %v ] \x1b[0m", num)
		////logger.Info("Finish: Agents %v\n", num)

	case api.MsgType_FORWARD_CLOCK_REQUEST:
		// クロックを進める要求
		forwardClock()

		// response
		senderId := myProvider.Id
		targets := []uint64{dm.GetSimDemand().GetSenderId()}
		msgId := dm.GetSimDemand().GetMsgId()
		simapi.ForwardClockResponse(senderId, targets, msgId)
		//logger.Info("Finish: Forward Clock")

	case api.MsgType_FORWARD_CLOCK_INIT_REQUEST:
		agentsMessage = NewMessage()

		// response
		senderId := myProvider.Id
		targets := []uint64{dm.GetSimDemand().GetSenderId()}
		msgId := dm.GetSimDemand().GetMsgId()
		simapi.ForwardClockInitResponse(senderId, targets, msgId)
		//logger.Info("Finish: Forward Clock Init")

	case api.MsgType_GET_AGENT_REQUEST:
		////logger.Debug("get agent request %v\n", dm)
		senderId := dm.GetSimDemand().GetSenderId()
		sameAreaIds := pm.GetProviderIds([]api.Provider_Type{
			api.Provider_SAME,
		})
		neighborAreaIds := pm.GetProviderIds([]api.Provider_Type{
			//api.Provider_NEIGHBOR,
			api.Provider_GATEWAY,
		})
		visIds := pm.GetProviderIds([]api.Provider_Type{
			api.Provider_VISUALIZATION,
		})

		agents := []*api.Agent{}
		if Contains(sameAreaIds, senderId) {
			// 同じエリアのエージェントプロバイダの場合
			agents = sim.Agents
		} else if Contains(neighborAreaIds, senderId) {
			// 隣接エリアのエージェントプロバイダの場合
			////logger.Debug("Get Agent Request from \n%v\n", dm)
			agents = agentsMessage.Get()
		} else if Contains(visIds, senderId) {
			// Visプロバイダの場合
			agents = agentsMessage.Get()
		}
		////logger.Debug("get agent request2 %v\n")

		// response
		pId := myProvider.Id
		targets := []uint64{dm.GetSimDemand().GetSenderId()}
		msgId := dm.GetSimDemand().GetMsgId()
		simapi.GetAgentResponse(pId, targets, msgId, agents)

	}
}

// callback for each Supply
func supplyCallback(clt *api.SMServiceClient, sp *api.Supply) {
	switch simMsg.GetType() {
	case api.MsgType_REGIST_PROVIDER_RESPONSE:
		//logger.Debug("resist provider response")
		mu.Lock()
		workerProvider = simMsg.GetRegistProviderResponse().GetProvider()
		mu.Unlock()
	case api.MsgType_GET_AGENT_RESPONSE:
		//time.Sleep(10 * time.Millisecond)
		////logger.Debug("get agent response \n", sp)
		simapi.SendSpToWait(sp)
	case api.MsgType_SET_AGENT_RESPONSE:
		////logger.Debug("response set agent")
		//time.Sleep(10 * time.Millisecond)
		////logger.Debug("get agent response \n", sp)
		simapi.SendSpToWait(sp)
	}
}*/

////////////////////////////////////////////////////////////
////////////         Worker Callback       ////////////////
///////////////////////////////////////////////////////////
type WorkerCallback struct {
	*util.Callback
}

func (cb *WorkerCallback) ForwardClockInitRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	logger.Debug("Got Forward Clock Init Request")
	// 同じエリアからエージェント情報を取得
	agentsMessage = NewMessage()
}

func (cb *WorkerCallback) ForwardClockMainRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	logger.Debug("Got Forward Clock Main Request")
	// エージェント情報を計算する
	forwardClock()
}

func (cb *WorkerCallback) ForwardClockTerminateRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	logger.Debug("Got Forward Clock Terminate Request")
}

func (cb *WorkerCallback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	logger.Debug("Got Update Providers Request")
	providers := simMsg.GetUpdateProvidersRequest().GetProviders()
	pm.SetProviders(providers)
}

func (cb *WorkerCallback) SetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	logger.Debug("Got Set Agent Request")
	// Agentをセットする
	agents := simMsg.GetSetAgentRequest().GetAgents()

	// Agent情報を追加する
	sim.AddAgents(agents)
}

func (cb *WorkerCallback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*api.Agent {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	logger.Debug("Got Get Agent Request")

	agents := sim.Agents
	return agents
}

////////////////////////////////////////////////////////////
////////////     Worker Demand Supply Callback     ////////
///////////////////////////////////////////////////////////

/*func MbcbClockWorker(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got clock callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	sclient := sclientOptsWorker[uint32(api.ChannelType_CLOCK)].Sclient
	switch simMsg.GetType() {
	case api.MsgType_FORWARD_CLOCK_INIT_REQUEST:
		logger.Debug("Got Forward Clock Init Request")
		// 同じエリアからエージェント情報を取得
		agentsMessage = NewMessage()

		// response
		//targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		simapi.ForwardClockInitResponse(sclient, msgId)
		//logger.Info("Finish: Forward Clock Init")
	case api.MsgType_FORWARD_CLOCK_MAIN_REQUEST:
		logger.Debug("Got Forward Clock Main Request")
		// エージェント情報を計算する
		forwardClock()

		// response
		//targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		simapi.ForwardClockResponse(sclient, msgId)
		//logger.Info("Finish: Forward Clock")
	case api.MsgType_FORWARD_CLOCK_TERMINATE_REQUEST:
		logger.Debug("Got Forward Clock Terminate Request")
		// 隣接エリアからエージェント情報を取得し更新

		// response
		//targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		simapi.ForwardClockResponse(sclient, msgId)
		//logger.Info("Finish: Forward Clock")
	}
}

func MbcbProviderWorker(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got provider callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_REGISTER_PROVIDER_RESPONSE:
		//logger.Debug("resist provider response")
		simapi.SendMsgToWait(msg)
	case api.MsgType_UPDATE_PROVIDERS_REQUEST:
		providers := simMsg.GetUpdateProvidersRequest().GetProviders()
		pm.SetProviders(providers)

		// response
		//targets := []uint64{simMsg.GetSenderId()}
		sclient := sclientOptsWorker[uint32(api.ChannelType_PROVIDER)].Sclient
		msgId := simMsg.GetMsgId()
		simapi.UpdateProvidersResponse(sclient, msgId)
		//logger.Info("Finish: Update Providers num: %v\n", len(providers))
		//for _, p := range providers {
		//logger.Debug("PID: %v,  Name: %v\n", p.Id, p.Name)
		//}
	}
}

func MbcbAgentWorker(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got agent callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	sclient := sclientOptsWorker[uint32(api.ChannelType_AGENT)].Sclient
	switch simMsg.GetType() {
	case api.MsgType_SET_AGENT_RESPONSE:
		simapi.SendMsgToWait(msg)
	case api.MsgType_GET_AGENT_RESPONSE:
		simapi.SendMsgToWait(msg)
	case api.MsgType_SET_AGENT_REQUEST:
		// Agentをセットする
		agents := simMsg.GetSetAgentRequest().GetAgents()

		// Agent情報を追加する
		sim.AddAgents(agents)

		// セット完了通知を送る
		//targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		simapi.SetAgentResponse(sclient, msgId)
		//logger.Info("\x1b[32m\x1b[40m [ Agent : %v ] \x1b[0m", num)
		////logger.Info("Finish: Agents %v\n", num)
	case api.MsgType_GET_AGENT_REQUEST:
		////logger.Debug("get agent request %v\n", dm)
		senderId := simMsg.GetSenderId()
		sameAreaIds := pm.GetProviderIds([]api.Provider_Type{
			api.Provider_AGENT,
		})
		neighborAreaIds := pm.GetProviderIds([]api.Provider_Type{
			//api.Provider_NEIGHBOR,
			api.Provider_GATEWAY,
		})
		visIds := pm.GetProviderIds([]api.Provider_Type{
			api.Provider_VISUALIZATION,
		})

		agents := []*api.Agent{}
		if Contains(sameAreaIds, senderId) {
			// 同じエリアのエージェントプロバイダの場合
			agents = sim.Agents
		} else if Contains(neighborAreaIds, senderId) {
			// 隣接エリアのエージェントプロバイダの場合
			////logger.Debug("Get Agent Request from \n%v\n", dm)
			agents = agentsMessage.Get()
		} else if Contains(visIds, senderId) {
			// Visプロバイダの場合
			agents = agentsMessage.Get()
		}
		////logger.Debug("get agent request2 %v\n")

		// response
		//targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		simapi.GetAgentResponse(sclient, msgId, agents)
	}
}

// 配列に値があるかどうか
func Contains(s []uint64, e uint64) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}

func MbcbAreaWorker(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got mbcb callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
}*/

////////////////////////////////////////////////////////////
////////////          Vis Callback         ////////////////
///////////////////////////////////////////////////////////
type VisCallback struct {
	*util.Callback
}

func (cb *VisCallback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*api.Agent {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	logger.Debug("Got Get Agent Request")

	agents := sim.Agents
	return agents
}

////////////////////////////////////////////////////////////
////////////    Vis Demand Supply Callback     ////////
///////////////////////////////////////////////////////////

/*func MbcbClockVis(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got clock callback")
}

func MbcbProviderVis(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got provider callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_REGISTER_PROVIDER_RESPONSE:
		simapi.SendMsgToWait(msg)
	}
}

func MbcbAgentVis(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got agent callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_SET_AGENT_RESPONSE:
		simapi.SendMsgToWait(msg)
	}
}

func MbcbAreaVis(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got mbcb callback")
}*/

//////////////////// for VIS ////////////////////////////
// callback for each Supply
/*func visDemandCallback(clt *api.SMServiceClient, dm *api.Demand) {

}

// callback for each Supply
func visSupplyCallback(clt *api.SMServiceClient, sp *api.Supply) {
	switch simMsg.GetType() {
	case api.MsgType_REGIST_PROVIDER_RESPONSE:
		//logger.Debug("resist provider response")
		mu.Lock()
		visProvider = simMsg.GetRegistProviderResponse().GetProvider()
		pm.AddProvider(visProvider)
		mu.Unlock()
	case api.MsgType_SET_AGENT_RESPONSE:
		////logger.Debug("response set agent from vis")
		//time.Sleep(10 * time.Millisecond)
		////logger.Debug("get agent response \n", sp)
		vissimapi.SendSpToWait(sp)
	}
}*/

func main() {
	//logger.Info("StartUp Provider")
	fmt.Printf("NumCPU=%d\n", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Printf("Start Worker Provider")

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
	logger.Info("Register Synerex Server")

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
	logger.Info("Register Synerex Server")

	wg := sync.WaitGroup{} // for syncing other goroutines
	wg.Add(1)

	// Simulator
	sim = NewSimulator(myArea, api.AgentType_PEDESTRIAN)

	time.Sleep(5 * time.Second)

	sclient := sclientOptsWorker[uint32(api.ChannelType_PROVIDER)].Sclient
	workerProvider = util.RegisterProviderLoop(sclient, simapi)
	logger.Info("Register Worker Provider")

	sclient = sclientOptsVis[uint32(api.ChannelType_PROVIDER)].Sclient
	visProvider = util.RegisterProviderLoop(sclient, simapi)
	logger.Info("Register Vis Provider")

	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!

}
