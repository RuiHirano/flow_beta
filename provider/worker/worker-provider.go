package main

import (
	"flag"
	"fmt"

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
	masterProvider *api.Provider
	mu             sync.Mutex
	workerClock    int
	logger         *util.Logger
	pm             *util.ProviderManager

	sclientOptsMaster map[uint32]*util.SclientOpt
	sclientOptsWorker map[uint32]*util.SclientOpt
	simapi            *api.SimAPI
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

	uid, _ := uuid.NewRandom()
	myProvider := &api.Provider{
		Id:   uint64(uid.ID()),
		Name: "WorkerProvider",
		Type: api.Provider_WORKER,
	}
	simapi = api.NewSimAPI(myProvider)
	pm = util.NewProviderManager(myProvider)
	logger.Info("ProviderID: %d", simapi.Provider.Id)

	cb := util.NewCallback()
	mscb := &MasterCallback{cb} // override
	sclientOptsMaster = map[uint32]*util.SclientOpt{
		uint32(api.ChannelType_CLOCK): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_CLOCK),
			MBusCallback: util.GetClockCallback(simapi, mscb),
			ArgJson:      fmt.Sprintf("{Client:WorkerProvider_Clock}"),
		},
		uint32(api.ChannelType_PROVIDER): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_PROVIDER),
			MBusCallback: util.GetProviderCallback(simapi, mscb),
			ArgJson:      fmt.Sprintf("{Client:WorkerProvider_Provider}"),
		},
		uint32(api.ChannelType_AGENT): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AGENT),
			MBusCallback: util.GetAgentCallback(simapi, mscb),
			ArgJson:      fmt.Sprintf("{Client:WorkerProvider_Agent}"),
		},
		uint32(api.ChannelType_AREA): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AREA),
			MBusCallback: util.GetAreaCallback(simapi, mscb),
			ArgJson:      fmt.Sprintf("{Client:WorkerProvider_Area}"),
		},
	}

	wocb := &WorkerCallback{cb} // override
	sclientOptsWorker = map[uint32]*util.SclientOpt{
		uint32(api.ChannelType_CLOCK): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_CLOCK),
			MBusCallback: util.GetClockCallback(simapi, wocb),
			ArgJson:      fmt.Sprintf("{Client:WorkerProvider_Clock}"),
		},
		uint32(api.ChannelType_PROVIDER): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_PROVIDER),
			MBusCallback: util.GetProviderCallback(simapi, wocb),
			ArgJson:      fmt.Sprintf("{Client:WorkerProvider_Provider}"),
		},
		uint32(api.ChannelType_AGENT): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AGENT),
			MBusCallback: util.GetAgentCallback(simapi, wocb),
			ArgJson:      fmt.Sprintf("{Client:WorkerProvider_Agent}"),
		},
		uint32(api.ChannelType_AREA): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AREA),
			MBusCallback: util.GetAreaCallback(simapi, wocb),
			ArgJson:      fmt.Sprintf("{Client:WorkerProvider_Area}"),
		},
	}

	workerClock = 0
	logger = util.NewLogger()
	logger.SetPrefix("Scenario")

}

////////////////////////////////////////////////////////////
////////////         Master Callback       ////////////////
///////////////////////////////////////////////////////////
type MasterCallback struct {
	*util.Callback
}

func (cb *MasterCallback) ForwardClockRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	t1 := time.Now()

	// request to worker providers
	/*targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	sclient := sclientOptsWorker[uint32(api.ChannelType_CLOCK)].Sclient
	// init
	simapi.ForwardClockInitRequest(sclient, targets)

	// main
	simapi.ForwardClockRequest(sclient, targets)

	// terminate
	simapi.ForwardClockTerminateRequest(sclient, targets)*/

	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("Forward Clock! Duration: %v ms, Wait: %d ms", duration, interval-duration)
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
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	filters := []*api.Filter{}
	for _, target := range targets {
		filters = append(filters, &api.Filter{TargetId: target})
	}
	sclient := sclientOptsWorker[uint32(api.ChannelType_AGENT)].Sclient
	simapi.SetAgentRequest(sclient, filters, agents)

	logger.Success("Set Agents Add: %v", len(agents))
}

func (cb *MasterCallback) SetClockRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)

	// request to providers
	clock := simMsg.GetSetClockRequest().GetClock()
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	filters := []*api.Filter{}
	for _, target := range targets {
		filters = append(filters, &api.Filter{TargetId: target})
	}
	sclient := sclientOptsWorker[uint32(api.ChannelType_CLOCK)].Sclient
	simapi.SetClockRequest(sclient, filters, clock)

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
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	p := simMsg.GetRegisterProviderRequest().GetProvider()
	pm.AddProvider(p)
	//fmt.Printf("regist provider! %v %v\n", p.GetId(), p.GetType())

	// update provider to worker
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_GATEWAY,
		api.Provider_AGENT,
	})
	filters := []*api.Filter{}
	for _, target := range targets {
		filters = append(filters, &api.Filter{TargetId: target})
	}
	sclient := sclientOptsWorker[uint32(api.ChannelType_PROVIDER)].Sclient
	//logger.Info("Send UpdateProvidersRequest %v, %v", targets, simapi.Provider)
	simapi.UpdateProvidersRequest(sclient, filters, pm.GetProviders())
	logger.Success("Update Providers! Worker Num: ", len(filters))

	return simapi.Provider
}

func main() {
	logger.Info("Start Worker Provider")
	logger.Info("NumCPU=%d", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Connect to Worker Syenrex Node Server
	// Register Node Server
	channelTypes := []uint32{}
	for _, opt := range sclientOptsWorker {
		channelTypes = append(channelTypes, opt.ChType)
	}
	ni := sxutil.GetDefaultNodeServInfo()
	util.RegisterNodeLoop(ni, *nodeaddr, "WorkerProvider", channelTypes)

	// Register Synerex Server
	client := util.RegisterSynerexLoop(*servaddr)
	util.RegisterSXServiceClients(ni, client, sclientOptsWorker)
	logger.Success("Subscribe Mbus")
	//logger.Info("Register Synerex Server")

	// Connect to Master Syenrex Node Server
	// Register Node Server
	channelTypes = []uint32{}
	for _, opt := range sclientOptsMaster {
		channelTypes = append(channelTypes, opt.ChType)
	}
	ni = sxutil.NewNodeServInfo()
	util.RegisterNodeLoop(ni, *masterNodeaddr, "WorkerProvider", channelTypes)

	// Register Synerex Server
	client = util.RegisterSynerexLoop(*masterServaddr)
	util.RegisterSXServiceClients(ni, client, sclientOptsMaster)
	logger.Success("Subscribe Mbus")

	wg := sync.WaitGroup{} // for syncing other goroutines
	wg.Add(1)
	sclient := sclientOptsMaster[uint32(api.ChannelType_PROVIDER)].Sclient
	//logger.Info("Register Master Provider %+v", sclientOptsMaster[uint32(api.ChannelType_PROVIDER)].Sclient)
	masterProvider = util.RegisterProviderLoop(sclient, simapi)
	logger.Success("Register Provider to Master Provider at %d", masterProvider.Id)

	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!
	logger.Success("Terminate Worker Provider")

}
