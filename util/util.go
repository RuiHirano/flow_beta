package util

import (
	"context"
	"log"
	"time"

	api "github.com/RuiHirano/flow_beta/api"
	"github.com/golang/protobuf/proto"
	sxapi "github.com/synerex/synerex_api"
	sxutil "github.com/synerex/synerex_sxutil"
)

type SclientOpt struct {
	Sclient      *sxutil.SXServiceClient
	ChType       uint32
	MBusCallback func(*sxutil.SXServiceClient, *sxapi.MbusMsg)
	ArgJson      string
	Providers    []*api.Provider
}

var (
	logger *Logger
)

func init() {
	logger = NewLogger()
}

// NodeServに繋がるまで繰り返す
func RegisterNodeLoop(ni *sxutil.NodeServInfo, nodesrv string, name string, chTypes []uint32) *sxutil.NodeServInfo {
	go sxutil.HandleSigInt() // Ctl+cを認識させる
	for {
		_, err := ni.RegisterNodeWithCmd(nodesrv, name, chTypes, nil, nil)
		if err != nil {
			logger.Error("Can't register node. reconeccting...")
			time.Sleep(1 * time.Second)
		} else {
			sxutil.RegisterDeferFunction(sxutil.UnRegisterNode)
			logger.Success("Connect NodeServer at [%s]", nodesrv)
			//ni := sxutil.GetDefaultNodeServInfo()
			return ni
		}
	}
}

func RegisterSXServiceClients(ni *sxutil.NodeServInfo, client sxapi.SynerexClient, opts map[uint32]*SclientOpt) map[uint32]*SclientOpt {
	for key, opt := range opts {
		sclient := ni.NewSXServiceClient(client, opt.ChType, opt.ArgJson) // service client
		sclient.MbusID = sxutil.IDType(opt.ChType)                        // MbusIDをChTypeに変更
		//log.Printf("debug MbusID: %d", sclient.MbusID)
		opts[key].Sclient = sclient
		go SubscribeMbusLoop(sclient, opt.MBusCallback)
	}
	return opts
}

func SubscribeMbusLoop(sclient *sxutil.SXServiceClient, mbcb func(*sxutil.SXServiceClient, *sxapi.MbusMsg)) {
	//called as goroutine
	ctx := context.Background() // should check proper context
	sxutil.RegisterDeferFunction(func() {
		logger.Info("Mbus Closing...")
		sclient.CloseMbus(ctx)
	})
	for {
		sclient.SubscribeMbus(ctx, mbcb)
		// comes here if channel closed
		logger.Error("SMarket Server Closed? Reconnecting...")
		time.Sleep(1 * time.Second)
	}
}

// Synerexに繋がるまで繰り返す
func RegisterSynerexLoop(sxServerAddress string) sxapi.SynerexClient {
	for {
		client := sxutil.GrpcConnectServer(sxServerAddress)
		if client == nil {
			logger.Error("Can't register synerex. reconeccting...")
			time.Sleep(1 * time.Second)
		} else {
			logger.Success("Connect Synerex Server at %s", sxServerAddress)
			return client
		}
	}
}

// WorkerやMasterにProviderを登録する
func RegisterProviderLoop(sclient *sxutil.SXServiceClient, simapi *api.SimAPI) *api.Provider {
	// masterへ登録
	filters := []*api.Filter{&api.Filter{
		TargetId: 0, // allow any msg
	}}
	//bc.simapi.RegisterProviderRequest(sclient, targets, bc.simapi.Provider)
	var provider *api.Provider
	ch := make(chan struct{})
	go func() {
		for {
			log.Printf("RegistProviderRequst %v", simapi.Provider.Id)
			msgs, err := simapi.RegisterProviderRequest(sclient, filters, simapi.Provider)
			if err != nil {
				//logger.Debug("Couldn't Regist Master...Retry...")
				logger.Error("Error: no response! recconect now... %v", err)
				time.Sleep(2 * time.Second)

			} else {
				//logger.Debug("Regist Success to Master!")
				provider = msgs[0].GetRegisterProviderResponse().GetProvider()
				logger.Success("Register Provider to %s", provider.Name)
				ch <- struct{}{}
				return
			}
		}
		return
	}()

	<-ch
	log.Printf("finish!")
	return provider
}

///////////////////////////////
// callback
/////////////////////////////

func GetAgentCallback(simapi *api.SimAPI, callback CallbackInterface) func(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	newCb := NewBaseCallback(simapi, callback)
	return newCb.AgentCallback
}

func GetProviderCallback(simapi *api.SimAPI, callback CallbackInterface) func(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	newCb := NewBaseCallback(simapi, callback)
	return newCb.ProviderCallback
}

func GetClockCallback(simapi *api.SimAPI, callback CallbackInterface) func(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	newCb := NewBaseCallback(simapi, callback)
	return newCb.ClockCallback
}

func GetAreaCallback(simapi *api.SimAPI, callback CallbackInterface) func(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	newCb := NewBaseCallback(simapi, callback)
	return newCb.ProviderCallback
}

type CallbackInterface interface {
	SetAgentRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	SetAgentResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	GetAgentRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg) []*api.Agent
	GetAgentResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	RegisterProviderRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg) *api.Provider
	RegisterProviderResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	UpdateProvidersRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	UpdateProvidersResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	SetClockRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	SetClockResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	StopClockRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	StopClockResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	StartClockRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	StartClockResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	ForwardClockRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	ForwardClockResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	ForwardClockInitRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	ForwardClockInitResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	ForwardClockMainRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	ForwardClockMainResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	ForwardClockTerminateRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	ForwardClockTerminateResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	SendAreaInfoRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	SendAreaInfoResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
}

type BaseCallback struct {
	simapi *api.SimAPI
	CallbackInterface
}

func NewBaseCallback(simapi *api.SimAPI, cbif CallbackInterface) *BaseCallback {
	bc := &BaseCallback{
		CallbackInterface: cbif,
		simapi:            simapi,
	}
	return bc
}

func (bc *BaseCallback) AgentCallback(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	//targets := []uint64{simMsg.GetSenderId()}
	log.Printf("get Agent Callback %v\n", simMsg)
	msgId := simMsg.GetMsgId()
	switch simMsg.GetType() {
	case api.MsgType_SET_AGENT_REQUEST:
		bc.SetAgentRequest(clt, msg)
		// response
		bc.simapi.SetAgentResponse(clt, msgId)
	case api.MsgType_GET_AGENT_REQUEST:
		agents := bc.GetAgentRequest(clt, msg)
		// response
		bc.simapi.GetAgentResponse(clt, msgId, agents)
	case api.MsgType_SET_AGENT_RESPONSE:
		bc.SetAgentResponse(clt, msg)
		bc.simapi.SendMsgToWait(msg)
	case api.MsgType_GET_AGENT_RESPONSE:
		bc.GetAgentResponse(clt, msg)
		bc.simapi.SendMsgToWait(msg)
	}
}

func (bc *BaseCallback) ProviderCallback(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	//targets := []uint64{simMsg.GetSenderId()}
	msgId := simMsg.GetMsgId()

	log.Printf("get Provider Callback %v\n", simMsg)
	switch simMsg.GetType() {
	case api.MsgType_REGISTER_PROVIDER_REQUEST:
		go func() {
			provider := bc.RegisterProviderRequest(clt, msg)
			// response
			bc.simapi.RegisterProviderResponse(clt, msgId, provider)
			return
		}()

	case api.MsgType_UPDATE_PROVIDERS_REQUEST:
		go func() {
			bc.UpdateProvidersRequest(clt, msg)
			// response
			bc.simapi.UpdateProvidersResponse(clt, msgId)
			return
		}()
	case api.MsgType_REGISTER_PROVIDER_RESPONSE:
		go func() {
			bc.RegisterProviderResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)
			return
		}()
	case api.MsgType_UPDATE_PROVIDERS_RESPONSE:
		go func() {
			bc.UpdateProvidersResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)
			return
		}()
	}
}

func (bc *BaseCallback) ClockCallback(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	//targets := []uint64{simMsg.GetSenderId()}
	log.Printf("get Clock Callback %v\n", simMsg)
	msgId := simMsg.GetMsgId()
	switch simMsg.GetType() {
	case api.MsgType_SET_CLOCK_REQUEST:
		bc.SetClockRequest(clt, msg)
		// response
		bc.simapi.SetClockResponse(clt, msgId)
	case api.MsgType_STOP_CLOCK_REQUEST:
		bc.StopClockRequest(clt, msg)
		// response
		bc.simapi.StopClockResponse(clt, msgId)
	case api.MsgType_START_CLOCK_REQUEST:
		bc.StartClockRequest(clt, msg)
		// response
		bc.simapi.StartClockResponse(clt, msgId)
	case api.MsgType_FORWARD_CLOCK_REQUEST:
		bc.ForwardClockRequest(clt, msg)
		// response
		bc.simapi.ForwardClockResponse(clt, msgId)
	case api.MsgType_FORWARD_CLOCK_INIT_REQUEST:
		bc.ForwardClockInitRequest(clt, msg)
		// response
		bc.simapi.ForwardClockInitResponse(clt, msgId)
	case api.MsgType_FORWARD_CLOCK_MAIN_REQUEST:
		bc.ForwardClockMainRequest(clt, msg)
		// response
		bc.simapi.ForwardClockMainResponse(clt, msgId)
	case api.MsgType_FORWARD_CLOCK_TERMINATE_REQUEST:
		bc.ForwardClockTerminateRequest(clt, msg)
		// response
		bc.simapi.ForwardClockTerminateResponse(clt, msgId)
	case api.MsgType_SET_CLOCK_RESPONSE:
		bc.SetClockResponse(clt, msg)
		bc.simapi.SendMsgToWait(msg)
	case api.MsgType_START_CLOCK_RESPONSE:
		bc.StartClockResponse(clt, msg)
		bc.simapi.SendMsgToWait(msg)
	case api.MsgType_STOP_CLOCK_RESPONSE:
		bc.StopClockResponse(clt, msg)
		bc.simapi.SendMsgToWait(msg)
	case api.MsgType_FORWARD_CLOCK_RESPONSE:
		bc.ForwardClockResponse(clt, msg)
		bc.simapi.SendMsgToWait(msg)
	case api.MsgType_FORWARD_CLOCK_INIT_RESPONSE:
		bc.ForwardClockInitResponse(clt, msg)
		bc.simapi.SendMsgToWait(msg)
	case api.MsgType_FORWARD_CLOCK_MAIN_RESPONSE:
		bc.ForwardClockMainResponse(clt, msg)
		bc.simapi.SendMsgToWait(msg)
	case api.MsgType_FORWARD_CLOCK_TERMINATE_RESPONSE:
		bc.ForwardClockTerminateResponse(clt, msg)
		bc.simapi.SendMsgToWait(msg)
	}
}

func (bc *BaseCallback) AreaCallback(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	//targets := []uint64{simMsg.GetSenderId()}
	msgId := simMsg.GetMsgId()
	log.Printf("get Area Callback %v\n", simMsg)
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_SEND_AREA_INFO_REQUEST:
		bc.SendAreaInfoRequest(clt, msg)
		// response
		bc.simapi.SendAreaInfoResponse(clt, msgId)
	case api.MsgType_SEND_AREA_INFO_RESPONSE:
		bc.SendAreaInfoResponse(clt, msg)
		bc.simapi.SendMsgToWait(msg)
	}
}

type Callback struct {
}

func NewCallback() *Callback {
	cb := &Callback{}
	return cb
}

// Agent
func (cb Callback) SetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)  {}
func (cb Callback) SetAgentResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {}
func (cb Callback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*api.Agent {
	return []*api.Agent{}
}
func (cb Callback) GetAgentResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {}

// Provider
func (cb Callback) RegisterProviderRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) *api.Provider {
	return &api.Provider{}
}
func (cb Callback) RegisterProviderResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {}
func (cb Callback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)   {}
func (cb Callback) UpdateProvidersResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)  {}

// Clock
func (cb Callback) SetClockRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)               {}
func (cb Callback) SetClockResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)              {}
func (cb Callback) StopClockRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)              {}
func (cb Callback) StopClockResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)             {}
func (cb Callback) StartClockRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)             {}
func (cb Callback) StartClockResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)            {}
func (cb Callback) ForwardClockRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)           {}
func (cb Callback) ForwardClockResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)          {}
func (cb Callback) ForwardClockInitRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)       {}
func (cb Callback) ForwardClockInitResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)      {}
func (cb Callback) ForwardClockMainRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)       {}
func (cb Callback) ForwardClockMainResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)      {}
func (cb Callback) ForwardClockTerminateRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)  {}
func (cb Callback) ForwardClockTerminateResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {}

// Area
func (cb Callback) SendAreaInfoRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)  {}
func (cb Callback) SendAreaInfoResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {}
