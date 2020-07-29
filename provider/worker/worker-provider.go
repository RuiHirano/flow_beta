package main

import (
	"flag"
	"fmt"
	"log"

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
	myProvider        *api.Provider
	masterProvider    *api.Provider
	workerSynerexAddr string
	workerNodeIdAddr  string
	masterNodeIdAddr  string
	masterSynerexAddr string
	providerName      string
	mu                sync.Mutex
	masterapi         *api.SimAPI
	workerapi         *api.SimAPI
	workerClock       int
	logger            *util.Logger
	pm                *util.ProviderManager

	sclientOptsMaster map[uint32]*util.SclientOpt
	sclientOptsWorker map[uint32]*util.SclientOpt
	simapi            *api.SimAPI
)

const MAX_AGENTS_NUM = 1000

func init() {
	uid, _ := uuid.NewRandom()
	myProvider := &api.Provider{
		Id:   uint64(uid.ID()),
		Name: "WorkerServer",
		Type: api.Provider_WORKER,
	}
	simapi = api.NewSimAPI(myProvider)
	pm = util.NewProviderManager(myProvider)
	log.Printf("ProviderID: %d", simapi.Provider.Id)

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
	flag.Parse()

	workerSynerexAddr = os.Getenv("SYNEREX_SERVER")
	if workerSynerexAddr == "" {
		workerSynerexAddr = "127.0.0.1:10000"
	}
	workerNodeIdAddr = os.Getenv("NODEID_SERVER")
	if workerNodeIdAddr == "" {
		workerNodeIdAddr = "127.0.0.1:9990"
	}
	masterSynerexAddr = os.Getenv("MASTER_SYNEREX_SERVER")
	if masterSynerexAddr == "" {
		masterSynerexAddr = "127.0.0.1:10000"
	}
	masterNodeIdAddr = os.Getenv("MASTER_NODEID_SERVER")
	if masterNodeIdAddr == "" {
		masterNodeIdAddr = "127.0.0.1:9990"
	}
	providerName = os.Getenv("PROVIDER_NAME")
	if providerName == "" {
		providerName = "WorkerProvider"
	}

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
	fmt.Printf("get forwardClockRequest")
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
	logger.Info("Duration: %v, PID: %v", duration, simapi.Provider.Id)
	// response to master
	//targets = []uint64{simMsg.GetSenderId()}
	//msgId := simMsg.GetMsgId()
	//logger.Debug("Response to master pid %v, msgId%v\n", myProvider.Id, msgId)
	//sclient = sclientOptsMaster[uint32(api.ChannelType_CLOCK)].Sclient
	//simapi.ForwardClockResponse(sclient, msgId)

}

func (cb *MasterCallback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	providers := simMsg.GetUpdateProvidersRequest().GetProviders()
	logger.Info("Finish: Update Workers num: %v\n", len(providers))
	//targets := []uint64{simMsg.GetSenderId()}
	//msgId := simMsg.GetMsgId()
	//sclient := sclientOptsMaster[uint32(api.ChannelType_PROVIDER)].Sclient
	//simapi.UpdateProvidersResponse(sclient, targets, msgId)
}

func (cb *MasterCallback) SetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)

	// request to providers
	agents := simMsg.GetSetAgentRequest().GetAgents()
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	sclient := sclientOptsWorker[uint32(api.ChannelType_AGENT)].Sclient
	simapi.SetAgentRequest(sclient, targets, agents)

	logger.Info("Finish: Set Agent %v\n")
}

////////////////////////////////////////////////////////////
////////////     Master Demand Supply Callback     ////////
///////////////////////////////////////////////////////////

/*func MbcbClockMaster(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got clock callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_FORWARD_CLOCK_REQUEST:
		fmt.Printf("get forwardClockRequest")
		t1 := time.Now()

		// request to worker providers
		targets := pm.GetProviderIds([]api.Provider_Type{
			api.Provider_AGENT,
		})
		sclient := sclientOptsWorker[uint32(api.ChannelType_CLOCK)].Sclient
		// init
		simapi.ForwardClockInitRequest(sclient, targets)

		// main
		simapi.ForwardClockRequest(sclient, targets)

		// terminate
		simapi.ForwardClockTerminateRequest(sclient, targets)

		t2 := time.Now()
		duration := t2.Sub(t1).Milliseconds()
		logger.Info("Duration: %v, PID: %v", duration, myProvider.Id)
		// response to master
		targets = []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		logger.Debug("Response to master pid %v, msgId%v\n", myProvider.Id, msgId)
		sclient = sclientOptsMaster[uint32(api.ChannelType_CLOCK)].Sclient
		simapi.ForwardClockResponse(sclient, targets, msgId)
	}
}

func MbcbProviderMaster(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got provider callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_REGISTER_PROVIDER_RESPONSE:
		simapi.SendMsgToWait(msg)
		logger.Info("regist provider to Master Provider!\n")
	case api.MsgType_UPDATE_PROVIDERS_REQUEST:
		providers := simMsg.GetUpdateProvidersRequest().GetProviders()
		targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		sclient := sclientOptsMaster[uint32(api.ChannelType_PROVIDER)].Sclient
		simapi.UpdateProvidersResponse(sclient, targets, msgId)
		logger.Info("Finish: Update Workers num: %v\n", len(providers))
	}
}

func MbcbAgentMaster(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got agent callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_SET_AGENT_REQUEST:
		fmt.Printf("set agent")
		// request to providers
		agents := simMsg.GetSetAgentRequest().GetAgents()
		targets := pm.GetProviderIds([]api.Provider_Type{
			api.Provider_AGENT,
		})
		sclient := sclientOptsWorker[uint32(api.ChannelType_AGENT)].Sclient
		simapi.SetAgentRequest(sclient, targets, agents)

		// response to master
		targets = []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		sclient = sclientOptsMaster[uint32(api.ChannelType_AGENT)].Sclient
		simapi.SetAgentResponse(sclient, targets, msgId)
	}
}

func MbcbAreaMaster(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got mbcb callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
}*/

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
	fmt.Printf("regist provider! %v %v\n", p.GetId(), p.GetType())

	// update provider to worker
	targets := pm.GetProviderIds([]api.Provider_Type{})
	//sclient := sclientOpts[uint32(api.ChannelType_PROVIDER)].Sclient
	logger.Info("Send UpdateProvidersRequest %v, %v", targets, simapi.Provider)
	//simapi.UpdateProvidersRequest(sclient, targets, pm.GetProviders())
	logger.Info("Success Update Providers! Worker Num: ", len(targets))

	return simapi.Provider
}

/*////////////////////////////////////////////////////////////
////////////     Worker Demand Supply Callback     ////////
///////////////////////////////////////////////////////////

func MbcbClockWorker(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got clock callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_SET_CLOCK_RESPONSE:
		simapi.SendMsgToWait(msg)
	case api.MsgType_FORWARD_CLOCK_INIT_RESPONSE:
		simapi.SendMsgToWait(msg)
	case api.MsgType_FORWARD_CLOCK_MAIN_RESPONSE:
		simapi.SendMsgToWait(msg)
	case api.MsgType_FORWARD_CLOCK_TERMINATE_RESPONSE:
		simapi.SendMsgToWait(msg)
	}
}

func MbcbProviderWorker(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got provider callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_UPDATE_PROVIDERS_RESPONSE:
		simapi.SendMsgToWait(msg)
	case api.MsgType_REGISTER_PROVIDER_REQUEST:
		// providerを追加する
		p := simMsg.GetRegisterProviderRequest().GetProvider()
		pm.AddProvider(p)
		fmt.Printf("regist request from agent of vis provider! %v\n", p)
		// 登録完了通知
		targets := []uint64{p.GetId()}
		msgId := simMsg.GetMsgId()
		sclient := sclientOptsWorker[uint32(api.ChannelType_PROVIDER)].Sclient
		simapi.RegisterProviderResponse(sclient, msgId, myProvider)

		logger.Info("Success Regist Agent or Vis Providers", targets)

		// 参加プロバイダの更新命令
		targets = pm.GetProviderIds([]api.Provider_Type{
			api.Provider_GATEWAY,
			api.Provider_AGENT,
		})
		providers := pm.GetProviders()
		simapi.UpdateProvidersRequest(sclient, targets, providers)
		logger.Info("Update Providers! Provider Num %v \n", len(targets))
	}
}

func MbcbAgentWorker(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got agent callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_SET_AGENT_RESPONSE:
		simapi.SendMsgToWait(msg)
	}
}

func MbcbAreaWorker(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got mbcb callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
}*/

func main() {
	fmt.Printf("NumCPU=%d\n", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Printf("Start Worker Provider")

	// Connect to Worker Syenrex Node Server
	// Register Node Server
	/*nodesrv := "127.0.0.1:9990"
	channelTypes := []uint32{}
	for _, opt := range sclientOptsWorker {
		channelTypes = append(channelTypes, opt.ChType)
	}
	util.RegisterNodeLoop(nodesrv, "WorkerProvider", channelTypes)

	// Register Synerex Server
	sxServerAddress := "127.0.0.1:10000"
	client := util.RegisterSynerexLoop(sxServerAddress)
	util.RegisterSXServiceClients(client, sclientOptsWorker)
	logger.Info("Register Synerex Server")*/

	// Connect to Master Syenrex Node Server
	// Register Node Server
	nodesrv := "127.0.0.1:9990"
	channelTypes := []uint32{}
	for _, opt := range sclientOptsMaster {
		channelTypes = append(channelTypes, opt.ChType)
	}
	util.RegisterNodeLoop(nodesrv, "WorkerProvider", channelTypes)

	// Register Synerex Server
	sxServerAddress := "127.0.0.1:10000"
	client := util.RegisterSynerexLoop(sxServerAddress)
	util.RegisterSXServiceClients(client, sclientOptsMaster)
	logger.Info("Register Synerex Server")

	wg := sync.WaitGroup{} // for syncing other goroutines
	wg.Add(1)
	sclient := sclientOptsMaster[uint32(api.ChannelType_PROVIDER)].Sclient
	//logger.Info("Register Master Provider %+v", sclientOptsMaster[uint32(api.ChannelType_PROVIDER)].Sclient)
	masterProvider = util.RegisterProviderLoop(sclient, simapi)
	logger.Info("Register Master Provider")

	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!

}
