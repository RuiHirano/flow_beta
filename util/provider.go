package util

import (
	"fmt"
	"sync"

	api "github.com/RuiHirano/flow_beta/api"
	sxutil "github.com/synerex/synerex_sxutil"
)

var (
	mu sync.Mutex
	//logger         *Logger
)

func init() {
}

type WorkerAPI struct {
	SimAPI *api.SimAPI
	ServerAddr string
	NodeAddr string
	SclientOpts map[uint32]*SclientOpt
}


func NewWorkerAPI(simapi *api.SimAPI, servAddr string, nodeAddr string, cb CallbackInterface ) *WorkerAPI {
	api := &WorkerAPI{
		SimAPI: simapi,
		ServerAddr: servAddr,
		NodeAddr: nodeAddr,
		SclientOpts: map[uint32]*SclientOpt{
			uint32(api.ChannelType_CLOCK): &SclientOpt{
				ChType:       uint32(api.ChannelType_CLOCK),
				MBusCallback: GetClockCallback(simapi, cb),
				ArgJson:      fmt.Sprintf("{Client:%s_Clock}", simapi.Provider.Name),
			},
			uint32(api.ChannelType_PROVIDER): &SclientOpt{
				ChType:       uint32(api.ChannelType_PROVIDER),
				MBusCallback: GetProviderCallback(simapi, cb),
				ArgJson:      fmt.Sprintf("{Client:%s_Provider}", simapi.Provider.Name),
			},
			uint32(api.ChannelType_AGENT): &SclientOpt{
				ChType:       uint32(api.ChannelType_AGENT),
				MBusCallback: GetAgentCallback(simapi, cb),
				ArgJson:      fmt.Sprintf("{Client:%s_Agent}", simapi.Provider.Name),
			},
			uint32(api.ChannelType_AREA): &SclientOpt{
				ChType:       uint32(api.ChannelType_AREA),
				MBusCallback: GetAreaCallback(simapi, cb),
				ArgJson:      fmt.Sprintf("{Client:%s_Clock}", simapi.Provider.Name),
			},
		},
	}
	return api
}

// Connect: Worker Nodeに接続する
func (ap *WorkerAPI) ConnectServer() error {
	channelTypes := []uint32{}
	for _, opt := range ap.SclientOpts {
		channelTypes = append(channelTypes, opt.ChType)
	}
	ni := sxutil.GetDefaultNodeServInfo()
	RegisterNodeLoop(ni, ap.NodeAddr, ap.SimAPI.Provider.Name, channelTypes)

	// Register Synerex Server
	client := RegisterSynerexLoop(ap.ServerAddr)

	// Register Callback
	RegisterSXServiceClients(ni, client, ap.SclientOpts)
	logger.Success("Finish Connect")
	return nil
}

func (ap *WorkerAPI) RegisterProvider() error {
	sclient := ap.SclientOpts[uint32(api.ChannelType_PROVIDER)].Sclient
	masterProvider := RegisterProviderLoop(sclient, ap.SimAPI)
	logger.Success("Register Provider to Worker Provider at %d", masterProvider.Id)
	return nil
}

func (ap *WorkerAPI) GetAgents(targets []uint64) []*api.Agent{
	sclient := ap.SclientOpts[uint32(api.ChannelType_AGENT)].Sclient
	agents := []*api.Agent{}
	simMsgs, _ := ap.SimAPI.GetAgentRequest(sclient, targets)
	for _, simMsg := range simMsgs {
		agents := simMsg.GetGetAgentResponse().GetAgents()
		agents = append(agents, agents...)
	}
	return agents
}

type MasterAPI struct {
	SimAPI *api.SimAPI
	ServerAddr string
	NodeAddr string
	SclientOpts map[uint32]*SclientOpt
}


func NewMasterAPI(simapi *api.SimAPI, servAddr string, nodeAddr string, cb CallbackInterface ) *MasterAPI {
	api := &MasterAPI{
		SimAPI: simapi,
		ServerAddr: servAddr,
		NodeAddr: nodeAddr,
		SclientOpts: map[uint32]*SclientOpt{
			uint32(api.ChannelType_CLOCK): &SclientOpt{
				ChType:       uint32(api.ChannelType_CLOCK),
				MBusCallback: GetClockCallback(simapi, cb),
				ArgJson:      fmt.Sprintf("{Client:%s_Clock}", simapi.Provider.Name),
			},
			uint32(api.ChannelType_PROVIDER): &SclientOpt{
				ChType:       uint32(api.ChannelType_PROVIDER),
				MBusCallback: GetProviderCallback(simapi, cb),
				ArgJson:      fmt.Sprintf("{Client:%s_Provider}", simapi.Provider.Name),
			},
			uint32(api.ChannelType_AGENT): &SclientOpt{
				ChType:       uint32(api.ChannelType_AGENT),
				MBusCallback: GetAgentCallback(simapi, cb),
				ArgJson:      fmt.Sprintf("{Client:%s_Agent}", simapi.Provider.Name),
			},
			uint32(api.ChannelType_AREA): &SclientOpt{
				ChType:       uint32(api.ChannelType_AREA),
				MBusCallback: GetAreaCallback(simapi, cb),
				ArgJson:      fmt.Sprintf("{Client:%s_Clock}", simapi.Provider.Name),
			},
		},
	}
	return api
}

// Connect: Worker Nodeに接続する
func (ap *MasterAPI) ConnectServer() error {
	channelTypes := []uint32{}
	for _, opt := range ap.SclientOpts {
		channelTypes = append(channelTypes, opt.ChType)
	}
	ni := sxutil.GetDefaultNodeServInfo()
	RegisterNodeLoop(ni, ap.NodeAddr, ap.SimAPI.Provider.Name, channelTypes)

	// Register Synerex Server
	client := RegisterSynerexLoop(ap.ServerAddr)

	// Register Callback
	RegisterSXServiceClients(ni, client, ap.SclientOpts)
	logger.Success("Finish Connect")
	return nil
}

func (ap *MasterAPI) RegisterProvider() error {
	sclient := ap.SclientOpts[uint32(api.ChannelType_PROVIDER)].Sclient
	masterProvider := RegisterProviderLoop(sclient, ap.SimAPI)
	logger.Success("Register Provider to Worker Provider at %d", masterProvider.Id)
	return nil
}

func (ap *MasterAPI) UpdateProviders(targets []uint64, providers []*api.Provider) {
	sclient := ap.SclientOpts[uint32(api.ChannelType_PROVIDER)].Sclient
	//logger.Info("Send UpdateProvidersRequest %v, %v", targets, simapi.Provider)
	ap.SimAPI.UpdateProvidersRequest(sclient, targets, providers)
}

func (ap *MasterAPI) SetAgents(targets []uint64, agents []*api.Agent) {
	sclient := ap.SclientOpts[uint32(api.ChannelType_AGENT)].Sclient
	ap.SimAPI.SetAgentRequest(sclient, targets, agents)
}

func (ap *MasterAPI) ForwardClockInit(targets []uint64) {
	sclient := ap.SclientOpts[uint32(api.ChannelType_CLOCK)].Sclient
	ap.SimAPI.ForwardClockInitRequest(sclient, targets)
}

func (ap *MasterAPI) ForwardClockMain(targets []uint64) {
	sclient := ap.SclientOpts[uint32(api.ChannelType_CLOCK)].Sclient
	ap.SimAPI.ForwardClockMainRequest(sclient, targets)
}

func (ap *MasterAPI) ForwardClockTerminate(targets []uint64) {
	sclient := ap.SclientOpts[uint32(api.ChannelType_CLOCK)].Sclient
	ap.SimAPI.ForwardClockTerminateRequest(sclient, targets)
}

func (ap *MasterAPI) SetClock(targets []uint64, clock *api.Clock) {
	sclient := ap.SclientOpts[uint32(api.ChannelType_CLOCK)].Sclient
	ap.SimAPI.SetClockRequest(sclient, targets, clock)
}

func (ap *MasterAPI) SetArea(targets []uint64) {
	//sclient := ap.SclientOpts[uint32(api.ChannelType_AGENT)].Sclient
	//ap.SimAPI.SetAreaRequest(sclient, targets, clock)
}