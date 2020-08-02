package main

// main synerex serverからgatewayを介してother synerex serverへ情報を送る
// 基本的に一方通行

import (
	"flag"
	"fmt"
	"log"
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
	apm             *AgentProviderManager
	waiter          *api.Waiter
	mu              sync.Mutex
	myProvider      *api.Provider
	workerProvider1 *api.Provider
	workerProvider2 *api.Provider
	agentProvider1  *api.Provider
	agentProvider2  *api.Provider
	logger          *util.Logger

	sclientOptsWorker1 map[uint32]*util.SclientOpt
	sclientOptsWorker2 map[uint32]*util.SclientOpt
	simapi             *api.SimAPI
	servaddr           = flag.String("servaddr", getServerAddress(), "The Synerex Server Listening Address")
	nodeaddr           = flag.String("nodeaddr", getNodeservAddress(), "Node ID Server Address")
	workerServaddr     = flag.String("workerServaddr", getWorkerServerAddress(), "Worker Synerex Server Listening Address")
	workerNodeaddr     = flag.String("workerNodeaddr", getWorkerNodeservAddress(), "Worker Node ID Server Address")
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

	uid, _ := uuid.NewRandom()
	myProvider := &api.Provider{
		Id:   uint64(uid.ID()),
		Name: "MasterServer",
		Type: api.Provider_MASTER,
	}
	simapi = api.NewSimAPI(myProvider)
	//pm = util.NewProviderManager(myProvider)
	log.Printf("ProviderID: %d", simapi.Provider.Id)

	cb := util.NewCallback()
	w1cb := &Worker1Callback{cb} // override
	sclientOptsWorker1 = map[uint32]*util.SclientOpt{
		uint32(api.ChannelType_CLOCK): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_CLOCK),
			MBusCallback: util.GetClockCallback(simapi, w1cb),
			ArgJson:      fmt.Sprintf("{Client:Worker1Provider_Clock}"),
		},
		uint32(api.ChannelType_PROVIDER): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_PROVIDER),
			MBusCallback: util.GetProviderCallback(simapi, w1cb),
			ArgJson:      fmt.Sprintf("{Client:Worker1Provider_Provider}"),
		},
		uint32(api.ChannelType_AGENT): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AGENT),
			MBusCallback: util.GetAgentCallback(simapi, w1cb),
			ArgJson:      fmt.Sprintf("{Client:Worker1Provider_Agent}"),
		},
		uint32(api.ChannelType_AREA): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AREA),
			MBusCallback: util.GetAreaCallback(simapi, w1cb),
			ArgJson:      fmt.Sprintf("{Client:Worker1Provider_Area}"),
		},
	}

	w2cb := &Worker2Callback{cb} // override
	sclientOptsWorker2 = map[uint32]*util.SclientOpt{
		uint32(api.ChannelType_CLOCK): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_CLOCK),
			MBusCallback: util.GetClockCallback(simapi, w2cb),
			ArgJson:      fmt.Sprintf("{Client:Worker2Provider_Clock}"),
		},
		uint32(api.ChannelType_PROVIDER): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_PROVIDER),
			MBusCallback: util.GetProviderCallback(simapi, w2cb),
			ArgJson:      fmt.Sprintf("{Client:Worker2Provider_Provider}"),
		},
		uint32(api.ChannelType_AGENT): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AGENT),
			MBusCallback: util.GetAgentCallback(simapi, w2cb),
			ArgJson:      fmt.Sprintf("{Client:Worker2Provider_Agent}"),
		},
		uint32(api.ChannelType_AREA): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AREA),
			MBusCallback: util.GetAreaCallback(simapi, w2cb),
			ArgJson:      fmt.Sprintf("{Client:Worker2Provider_Area}"),
		},
	}

	//flag.Parse()
	logger = util.NewLogger()
	waiter = api.NewWaiter()
	apm = NewAgentProviderManager()

}

////////////////////////////////////////////////////////////
//////////         Agent Provider Manager         /////////
///////////////////////////////////////////////////////////

type AgentProviderManager struct {
	Provider1 *api.Provider
	Provider2 *api.Provider
}

func NewAgentProviderManager() *AgentProviderManager {
	apm := &AgentProviderManager{
		Provider1: nil,
		Provider2: nil,
	}
	return apm
}

func (apm *AgentProviderManager) SetProvider1(ps []*api.Provider) {
	mu.Lock()
	for _, p := range ps {
		logger.Debug("Provider1: PID: %v, Name: %v\n", p.Id, p.Name)
		if apm.Provider1 == nil && p.GetType() == api.Provider_AGENT {
			apm.Provider1 = p
			logger.Warn("Set Provider1!\n")
		}
	}
	mu.Unlock()
}

func (apm *AgentProviderManager) SetProvider2(ps []*api.Provider) {
	mu.Lock()
	for _, p := range ps {
		logger.Debug("Provider2: PID: %v, Name: %v\n", p.Id, p.Name)
		if apm.Provider2 == nil && p.GetType() == api.Provider_AGENT {
			apm.Provider2 = p
			logger.Warn("Set Provider2!\n")
		}
	}
	mu.Unlock()
}

/*type AgentProviderManager struct {
	Providers1  []*api.Provider
	Providers2  []*api.Provider
	NeighborMap map[uint64][]*api.Provider // 隣接してるProviderマップ
	MsgIdMap    map[uint64]uint64          // msgIdを結びつけるためのマップ
}

func NewAgentProviderManager() *AgentProviderManager {
	apm := &AgentProviderManager{
		Providers1:  []*api.Provider{},
		Providers2:  []*api.Provider{},
		NeighborMap: make(map[uint64][]*api.Provider),
		MsgIdMap:    make(map[uint64]uint64),
	}
	return apm
}

func (apm *AgentProviderManager) SetProviders1(ps []*api.Provider) {
	mu.Lock()
	for _, p := range ps {
		if p.GetProviderType() == api.Provider_AGENT {
			apm.Providers1 = append(apm.Providers1, p)
		}
	}
	apm.CreateProvidersMap()
	mu.Unlock()
}

func (apm *AgentProviderManager) SetProviders2(ps []*api.Provider) {
	mu.Lock()
	apm.Providers2 = []*api.Provider{}
	for _, p := range ps {
		if p.GetProviderType() == api.Provider_AGENT {
			apm.Providers2 = append(apm.Providers2, p)
		}
	}
	apm.CreateProvidersMap()
	mu.Unlock()
}

func (apm *AgentProviderManager) SetMsgIdMap(msgId1 uint64, msgId2 uint64) {
	mu.Lock()
	apm.MsgIdMap[msgId1] = msgId2
	apm.MsgIdMap[msgId2] = msgId1
	mu.Unlock()
}

func (apm *AgentProviderManager) CreateProvidersMap() {
	neighborMap := make(map[uint64][]*api.Provider)
	for _, p1 := range apm.Providers1 {
		p1Id := p1.GetId()
		for _, p2 := range apm.Providers2 {
			p2Id := p2.GetId()
			//if isNeighborArea(p1, p2) {
			// エリアが隣接していた場合
			neighborMap[p1Id] = append(neighborMap[p1Id], p2)
			neighborMap[p2Id] = append(neighborMap[p2Id], p1)
			//}
		}
	}
	apm.NeighborMap = neighborMap
}*/

/*func isNeighborArea(p1 *api.Provider, p2 *api.Provider) bool {
	myControlArea := pm.MyProvider.GetAgentStatus().GetArea().GetControlArea()
	tControlArea := p.GetAgentStatus().GetArea().GetControlArea()
	maxLat, maxLon, minLat, minLon := GetCoordRange(myControlArea)
	tMaxLat, tMaxLon, tMinLat, tMinLon := GetCoordRange(tControlArea)
	if maxLat == tMinLat && (minLon <= tMaxLon && tMaxLon <= maxLon || minLon <= tMinLon && tMinLon <= maxLon) {
		return true
	}
	if minLat == tMaxLat && (minLon <= tMaxLon && tMaxLon <= maxLon || minLon <= tMinLon && tMinLon <= maxLon) {
		return true
	}
	if maxLon == tMinLon && (minLat <= tMaxLat && tMaxLat <= maxLat || minLat <= tMinLat && tMinLat <= maxLat) {
		return true
	}
	if minLon == tMaxLon && (minLat <= tMaxLat && tMaxLat <= maxLat || minLat <= tMinLat && tMinLat <= maxLat) {
		return true
	}
	return false
}*/

////////////////////////////////////////////////////////////
//////////     Worker1 Demand Supply Callback     /////////
///////////////////////////////////////////////////////////

// Supplyのコールバック関数
/*func supplyCallback1(clt *api.SMServiceClient, sp *api.Supply) {
	switch sp.GetSimSupply().GetType() {
	case api.SupplyType_READY_PROVIDER_RESPONSE:
		//time.Sleep(10 * time.Millisecond)
		worker1api.SendSpToWait(sp)
		fmt.Printf("ready provider response")

	case api.SupplyType_GET_AGENT_RESPONSE:
		fmt.Printf("Get Sp from Worker1\n")

		//time.Sleep(10 * time.Millisecond)
		worker1api.SendSpToWait(sp)

	case api.SupplyType_REGIST_PROVIDER_RESPONSE:
		mu.Lock()
		worker1 = sp.GetSimSupply().GetRegistProviderResponse().GetProvider()
		mu.Unlock()
		fmt.Printf("regist provider to Worler1 Provider!\n")
	}
}

// Demandのコールバック関数
func demandCallback1(clt *api.SMServiceClient, dm *api.Demand) {

	switch dm.GetSimDemand().GetType() {

	case api.DemandType_GET_AGENT_REQUEST:
		logger.Debug("get agent request\n")
		// 隣接エリアがない場合はそのまま返す
		t1 := time.Now()

		agents := []*api.Agent{}
		senderId := myProvider.Id
		// worker2のagent-providerから取得
		targets2 := []uint64{agentProvider2.Id}
		sps, _ := worker2api.GetAgentRequest(senderId, targets2)

		for _, sp := range sps {
			ags := sp.GetSimSupply().GetGetAgentResponse().GetAgents()
			agents = append(agents, ags...)
		}

		targets := []uint64{dm.GetSimDemand().GetSenderId()}
		msgId := dm.GetSimDemand().GetMsgId()
		worker1api.GetAgentResponse(senderId, targets, msgId, agents)
		logger.Debug("Finish: Get Agent Response to Worker1 %v %v %v\n", targets, msgId)

		t2 := time.Now()
		duration := t2.Sub(t1).Milliseconds()
		logger.Info("Duration: %v, PID: %v", duration, myProvider.Id)

	case api.DemandType_UPDATE_PROVIDERS_REQUEST:
		logger.Debug("update providers request\n")
		ps1 := dm.GetSimDemand().GetUpdateProvidersRequest().GetProviders()

		//test
		for _, p := range ps1 {
			if agentProvider1 == nil && p.GetType() == api.Provider_AGENT {
				mu.Lock()
				agentProvider1 = p
				mu.Unlock()
				logger.Debug("Set Provider1!\n")
			}
		}

		// response
		targets := []uint64{dm.GetSimDemand().GetSenderId()}
		senderId := myProvider.Id
		msgId := dm.GetSimDemand().GetMsgId()
		worker1api.UpdateProvidersResponse(senderId, targets, msgId)
		logger.Info("Finish: Update Providers1 num: %v\n", len(ps1))
	}
}*/

////////////////////////////////////////////////////////////
////////////         Worker1 Callback       ////////////////
///////////////////////////////////////////////////////////

type Worker1Callback struct {
	*util.Callback
}

func (cb *Worker1Callback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	logger.Debug("update providers request\n")
	ps1 := simMsg.GetUpdateProvidersRequest().GetProviders()
	//test
	for _, p := range ps1 {
		if agentProvider1 == nil && p.GetType() == api.Provider_AGENT {
			mu.Lock()
			agentProvider1 = p
			mu.Unlock()
			logger.Debug("Set Provider1!\n")
		}
	}
}

func (cb *Worker1Callback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*api.Agent {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	logger.Debug("get agent request\n")
	// 隣接エリアがない場合はそのまま返す
	t1 := time.Now()

	agents := []*api.Agent{}
	// worker2のagent-providerから取得
	targets := []uint64{agentProvider2.Id}
	filters := []*api.Filter{}
	for _, target := range targets {
		filters = append(filters, &api.Filter{TargetId: target})
	}
	sclient := sclientOptsWorker2[uint32(api.ChannelType_AGENT)].Sclient
	msgs, _ := simapi.GetAgentRequest(sclient, filters)

	for _, msg := range msgs {
		ags := msg.GetGetAgentResponse().GetAgents()
		agents = append(agents, ags...)
	}
	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	logger.Info("Duration: %v, PID: %v", duration, simapi.Provider.Id)
	return agents
}

////////////////////////////////////////////////////////////
////////////     Worker1 Demand Supply Callback     ////////
///////////////////////////////////////////////////////////

/*func MbcbClockWorker1(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got clock callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
}

func MbcbProviderWorker1(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got provider callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	sclient := sclientOptsWorker1[uint32(api.ChannelType_PROVIDER)].Sclient
	switch simMsg.GetType() {
	case api.MsgType_UPDATE_PROVIDERS_REQUEST:
		logger.Debug("update providers request\n")
		ps1 := simMsg.GetUpdateProvidersRequest().GetProviders()
		//test
		for _, p := range ps1 {
			if agentProvider1 == nil && p.GetType() == api.Provider_AGENT {
				mu.Lock()
				agentProvider1 = p
				mu.Unlock()
				logger.Debug("Set Provider1!\n")
			}
		}
		// response
		targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		simapi.UpdateProvidersResponse(sclient, targets, msgId)
		logger.Info("Finish: Update Providers1 num: %v\n", len(ps1))
	case api.MsgType_REGISTER_PROVIDER_RESPONSE:
		simapi.SendMsgToWait(msg)
		fmt.Printf("regist provider to Worler1 Provider!\n")
	}
}

func MbcbAgentWorker1(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got agent callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_GET_AGENT_RESPONSE:
		fmt.Printf("Get Sp from Worker1\n")
		simapi.SendMsgToWait(msg)
	case api.MsgType_GET_AGENT_REQUEST:
		logger.Debug("get agent request\n")
		// 隣接エリアがない場合はそのまま返す
		t1 := time.Now()

		agents := []*api.Agent{}
		// worker2のagent-providerから取得
		targets2 := []uint64{agentProvider2.Id}
		sclient := sclientOptsWorker2[uint32(api.ChannelType_AGENT)].Sclient
		msgs, _ := simapi.GetAgentRequest(sclient, targets2)

		for _, msg := range msgs {
			ags := msg.GetGetAgentResponse().GetAgents()
			agents = append(agents, ags...)
		}

		targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		sclient = sclientOptsWorker1[uint32(api.ChannelType_AGENT)].Sclient
		simapi.GetAgentResponse(sclient, targets, msgId, agents)
		logger.Debug("Finish: Get Agent Response to Worker1 %v %v %v\n", targets, msgId)

		t2 := time.Now()
		duration := t2.Sub(t1).Milliseconds()
		logger.Info("Duration: %v, PID: %v", duration, myProvider.Id)
	}
}

func MbcbAreaWorker1(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got mbcb callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
}*/

///////////////////////////////////////////////////////////
////////////         Worker1 Callback       ////////////////
///////////////////////////////////////////////////////////

type Worker2Callback struct {
	*util.Callback
}

func (cb *Worker2Callback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	logger.Debug("update providers request\n")
	ps2 := simMsg.GetUpdateProvidersRequest().GetProviders()
	//test
	for _, p := range ps2 {
		if agentProvider2 == nil && p.GetType() == api.Provider_AGENT {
			mu.Lock()
			agentProvider2 = p
			mu.Unlock()
			logger.Debug("Set Provider1!\n")
		}
	}
}

func (cb *Worker2Callback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*api.Agent {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	logger.Debug("get agent request\n")
	// 隣接エリアがない場合はそのまま返す
	t1 := time.Now()

	agents := []*api.Agent{}
	// worker2のagent-providerから取得
	targets := []uint64{agentProvider2.Id}
	filters := []*api.Filter{}
	for _, target := range targets {
		filters = append(filters, &api.Filter{TargetId: target})
	}
	sclient := sclientOptsWorker1[uint32(api.ChannelType_AGENT)].Sclient
	msgs, _ := simapi.GetAgentRequest(sclient, filters)

	for _, msg := range msgs {
		ags := msg.GetGetAgentResponse().GetAgents()
		agents = append(agents, ags...)
	}

	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	logger.Info("Duration: %v, PID: %v", duration, myProvider.Id)
	return agents
}

////////////////////////////////////////////////////////////
////////////     Worker2 Demand Supply Callback     ////////
///////////////////////////////////////////////////////////

/*func MbcbClockWorker2(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got clock callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
}

func MbcbProviderWorker2(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got provider callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	sclient := sclientOptsWorker2[uint32(api.ChannelType_PROVIDER)].Sclient
	switch simMsg.GetType() {
	case api.MsgType_UPDATE_PROVIDERS_REQUEST:
		logger.Debug("update providers request\n")
		ps2 := simMsg.GetUpdateProvidersRequest().GetProviders()
		//test
		for _, p := range ps2 {
			if agentProvider2 == nil && p.GetType() == api.Provider_AGENT {
				mu.Lock()
				agentProvider2 = p
				mu.Unlock()
				logger.Debug("Set Provider1!\n")
			}
		}
		// response
		targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		simapi.UpdateProvidersResponse(sclient, targets, msgId)
		logger.Info("Finish: Update Providers1 num: %v\n", len(ps2))
	case api.MsgType_REGISTER_PROVIDER_RESPONSE:
		simapi.SendMsgToWait(msg)
		fmt.Printf("regist provider to Worler1 Provider!\n")
	}
}

func MbcbAgentWorker2(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got agent callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_GET_AGENT_RESPONSE:
		fmt.Printf("Get Sp from Worker2\n")
		simapi.SendMsgToWait(msg)
	case api.MsgType_GET_AGENT_REQUEST:
		logger.Debug("get agent request\n")
		// 隣接エリアがない場合はそのまま返す
		t1 := time.Now()

		agents := []*api.Agent{}
		// worker2のagent-providerから取得
		targets2 := []uint64{agentProvider2.Id}
		sclient := sclientOptsWorker1[uint32(api.ChannelType_AGENT)].Sclient
		msgs, _ := simapi.GetAgentRequest(sclient, targets2)

		for _, msg := range msgs {
			ags := msg.GetGetAgentResponse().GetAgents()
			agents = append(agents, ags...)
		}

		targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		sclient = sclientOptsWorker2[uint32(api.ChannelType_AGENT)].Sclient
		simapi.GetAgentResponse(sclient, targets, msgId, agents)
		logger.Debug("Finish: Get Agent Response to Worker2 %v %v %v\n", targets, msgId)

		t2 := time.Now()
		duration := t2.Sub(t1).Milliseconds()
		logger.Info("Duration: %v, PID: %v", duration, myProvider.Id)
	}
}

func MbcbAreaWorker2(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got mbcb callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
}*/

////////////////////////////////////////////////////////////
//////////     Worker2 Demand Supply Callback     /////////
///////////////////////////////////////////////////////////

// Supplyのコールバック関数
/*func supplyCallback2(clt *api.SMServiceClient, sp *api.Supply) {
	switch sp.GetSimSupply().GetType() {
	case api.SupplyType_GET_AGENT_RESPONSE:
		fmt.Printf("Get Sp from Worker2\n")
		worker2api.SendSpToWait(sp)

	case api.SupplyType_REGIST_PROVIDER_RESPONSE:
		mu.Lock()
		worker2 = sp.GetSimSupply().GetRegistProviderResponse().GetProvider()
		mu.Unlock()
		fmt.Printf("regist provider to Worler2 Provider!\n")

	case api.SupplyType_READY_PROVIDER_RESPONSE:
		//time.Sleep(10 * time.Millisecond)
		worker2api.SendSpToWait(sp)
		fmt.Printf("ready provider response")
	}
}

// Demandのコールバック関数
func demandCallback2(clt *api.SMServiceClient, dm *api.Demand) {
	switch dm.GetSimDemand().GetType() {

	case api.DemandType_GET_AGENT_REQUEST:
		logger.Debug("get agent request\n", dm.)

		t1 := time.Now()
		// 隣接エリアがない場合はそのまま返す
		agents := []*api.Agent{}
		senderId := myProvider.Id
		// worker2のagent-providerから取得
		targets1 := []uint64{agentProvider1.Id}
		sps, _ := worker1api.GetAgentRequest(senderId, targets1)

		for _, sp := range sps {
			ags := sp.GetSimSupply().GetGetAgentResponse().GetAgents()
			agents = append(agents, ags...)
		}

		targets := []uint64{dm.GetSimDemand().GetSenderId()}
		msgId := dm.GetSimDemand().GetMsgId()
		worker2api.GetAgentResponse(senderId, targets, msgId, agents)
		logger.Debug("Finish: Get Agent Response to Worker2 %v %v\n", targets, msgId)

		t2 := time.Now()
		duration := t2.Sub(t1).Milliseconds()
		logger.Info("Duration: %v, PID: %v", duration, myProvider.Id)
		// ない場合はそのまま返す

	case api.DemandType_UPDATE_PROVIDERS_REQUEST:
		logger.Debug("update providers request\n")
		ps2 := dm.GetSimDemand().GetUpdateProvidersRequest().GetProviders()

		//test
		for _, p := range ps2 {
			if agentProvider2 == nil && p.GetType() == api.Provider_AGENT {
				mu.Lock()
				agentProvider2 = p
				mu.Unlock()
				logger.Debug("Set Provider2!\n")
			}
		}

		// response
		targets := []uint64{dm.GetSimDemand().GetSenderId()}
		senderId := myProvider.Id
		msgId := dm.GetSimDemand().GetMsgId()
		worker2api.UpdateProvidersResponse(senderId, targets, msgId)
		logger.Info("Finish: Update Providers2 num: %v\n", len(ps2))
	}

}*/

func main() {
	logger.Info("StartUp Provider")
	fmt.Printf("NumCPU=%d\n", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Printf("Start Gateway Provider")

	// Connect to Worker Syenrex Node Server
	// Register Node Server
	channelTypes := []uint32{}
	for _, opt := range sclientOptsWorker1 {
		channelTypes = append(channelTypes, opt.ChType)
	}
	ni := sxutil.GetDefaultNodeServInfo()
	util.RegisterNodeLoop(ni, *nodeaddr, "GatewayProvider", channelTypes)

	// Register Synerex Server
	client := util.RegisterSynerexLoop(*servaddr)
	util.RegisterSXServiceClients(ni, client, sclientOptsWorker1)
	logger.Info("Register Synerex Server")

	// Connect to Master Syenrex Node Server
	// Register Node Server

	channelTypes = []uint32{}
	for _, opt := range sclientOptsWorker2 {
		channelTypes = append(channelTypes, opt.ChType)
	}
	ni = sxutil.NewNodeServInfo()
	util.RegisterNodeLoop(ni, *workerNodeaddr, "GatewayProvider", channelTypes)

	// Register Synerex Server
	client = util.RegisterSynerexLoop(*workerServaddr)
	util.RegisterSXServiceClients(ni, client, sclientOptsWorker2)
	logger.Info("Register Synerex Server")

	wg := sync.WaitGroup{} // for syncing other goroutines
	wg.Add(1)

	sclient := sclientOptsWorker1[uint32(api.ChannelType_PROVIDER)].Sclient
	workerProvider1 = util.RegisterProviderLoop(sclient, simapi)
	logger.Info("Register Worker1 Provider")

	sclient = sclientOptsWorker2[uint32(api.ChannelType_PROVIDER)].Sclient
	workerProvider2 = util.RegisterProviderLoop(sclient, simapi)
	logger.Info("Register Worker2 Provider")

	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!

}
