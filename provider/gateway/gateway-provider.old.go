package main

// main synerex serverからgatewayを介してother synerex serverへ情報を送る
// 基本的に一方通行

import (
	"flag"
	"fmt"

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
	flag.Parse()
	uid, _ := uuid.NewRandom()
	myProvider := &api.Provider{
		Id:   uint64(uid.ID()),
		Name: *providerName,
		Type: api.Provider_GATEWAY,
	}
	simapi = api.NewSimAPI(myProvider)
	//pm = util.NewProviderManager(myProvider)
	logger.Info("ProviderID: %d", simapi.Provider.Id)

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
		//logger.Debug("Provider1: PID: %v, Name: %v\n", p.Id, p.Name)
		if apm.Provider1 == nil && p.GetType() == api.Provider_AGENT {
			apm.Provider1 = p
			//logger.Warn("Set Provider1!\n")
		}
	}
	mu.Unlock()
}

func (apm *AgentProviderManager) SetProvider2(ps []*api.Provider) {
	mu.Lock()
	for _, p := range ps {
		//logger.Debug("Provider2: PID: %v, Name: %v\n", p.Id, p.Name)
		if apm.Provider2 == nil && p.GetType() == api.Provider_AGENT {
			apm.Provider2 = p
			//logger.Warn("Set Provider2!\n")
		}
	}
	mu.Unlock()
}

////////////////////////////////////////////////////////////
////////////         Worker1 Callback       ////////////////
///////////////////////////////////////////////////////////

type Worker1Callback struct {
	*util.Callback
}

func (cb *Worker1Callback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	//logger.Debug("update providers request\n")
	ps1 := simMsg.GetUpdateProvidersRequest().GetProviders()
	//test
	for _, p := range ps1 {
		if agentProvider1 == nil && p.GetType() == api.Provider_AGENT {
			mu.Lock()
			agentProvider1 = p
			mu.Unlock()
			logger.Success("Update Agent Id: %d\n", agentProvider1.Id)
		}
	}
}

func (cb *Worker1Callback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*api.Agent {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	agents := []*api.Agent{}
	if simMsg.GetSenderId() == agentProvider1.Id {
		//logger.Debug("get agent request 0\n")
		// 隣接エリアがない場合はそのまま返す
		t1 := time.Now()

		// worker2のagent-providerから取得
		targets := []uint64{agentProvider2.Id}
		filters := []*api.Filter{}
		for _, target := range targets {
			filters = append(filters, &api.Filter{TargetId: target})
		}
		//logger.Debug("get agent request 1\n")
		sclient := sclientOptsWorker2[uint32(api.ChannelType_AGENT)].Sclient
		msgs, _ := simapi.GetAgentRequest(sclient, filters)
		//logger.Debug("get agent request 2\n")
		for _, msg := range msgs {
			ags := msg.GetGetAgentResponse().GetAgents()
			agents = append(agents, ags...)
		}
		//logger.Debug("get agent request 3\n")
		t2 := time.Now()
		duration := t2.Sub(t1).Milliseconds()
		interval := int64(1000) // 周期ms
		if duration > interval {
			logger.Warn("time cycle delayed... Duration: %d", duration)
		} else {
			logger.Success("Get Agent! Duration: %v ms, Wait: %d ms", duration, interval-duration)
		}
	}
	return agents
}

///////////////////////////////////////////////////////////
////////////         Worker1 Callback       ////////////////
///////////////////////////////////////////////////////////

type Worker2Callback struct {
	*util.Callback
}

func (cb *Worker2Callback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	//logger.Debug("update providers request\n")
	ps2 := simMsg.GetUpdateProvidersRequest().GetProviders()
	//test
	for _, p := range ps2 {
		if agentProvider2 == nil && p.GetType() == api.Provider_AGENT {
			mu.Lock()
			agentProvider2 = p
			mu.Unlock()
			logger.Success("Update Agent Id: %d\n", agentProvider2.Id)
		}
	}
}

func (cb *Worker2Callback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*api.Agent {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	agents := []*api.Agent{}
	if simMsg.GetSenderId() == agentProvider2.Id {
		//logger.Warn("get agent request 0\n")
		// 隣接エリアがない場合はそのまま返す
		t1 := time.Now()

		// worker2のagent-providerから取得
		targets := []uint64{agentProvider1.Id}
		filters := []*api.Filter{}
		for _, target := range targets {
			filters = append(filters, &api.Filter{TargetId: target})
		}
		//logger.Warn("get agent request 1\n")
		sclient := sclientOptsWorker1[uint32(api.ChannelType_AGENT)].Sclient
		msgs, _ := simapi.GetAgentRequest(sclient, filters)
		//logger.Warn("get agent request 2\n")

		for _, msg := range msgs {
			ags := msg.GetGetAgentResponse().GetAgents()
			agents = append(agents, ags...)
		}
		//logger.Warn("get agent request 3\n")
		t2 := time.Now()
		duration := t2.Sub(t1).Milliseconds()
		interval := int64(1000) // 周期ms
		if duration > interval {
			logger.Warn("time cycle delayed... Duration: %d", duration)
		} else {
			logger.Success("Get Agent! Duration: %v ms, Wait: %d ms", duration, interval-duration)
		}
	}
	return agents
}

func main() {
	logger.Info("Start Gateway Provider")
	logger.Info("NumCPU=%d", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())

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
	logger.Success("Subscribe Mbus")

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
	logger.Success("Subscribe Mbus")

	wg := sync.WaitGroup{} // for syncing other goroutines
	wg.Add(1)

	sclient := sclientOptsWorker1[uint32(api.ChannelType_PROVIDER)].Sclient
	workerProvider1 = util.RegisterProviderLoop(sclient, simapi)
	logger.Success("Register Provider to Worker1 Provider at %d", workerProvider1.Id)

	sclient = sclientOptsWorker2[uint32(api.ChannelType_PROVIDER)].Sclient
	workerProvider2 = util.RegisterProviderLoop(sclient, simapi)
	logger.Success("Register Provider to Worker2 Provider at %d", workerProvider2.Id)

	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!
	logger.Success("Terminate Agent Provider")

}
