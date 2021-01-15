package api

import (
	"context"
	fmt "fmt"
	"time"

	//"log"

	"log"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	sxapi "github.com/synerex/synerex_api"
	sxutil "github.com/synerex/synerex_sxutil"
)

var (
	mu sync.Mutex
)

func init() {
}

////////////////////////////////////////////////////////////
//////////////       Provider Manager Class      //////////
///////////////////////////////////////////////////////////

type ProviderManager struct {
	MyProvider   *Provider
	Providers    []*Provider
	ProvidersMap map[Provider_Type][]*Provider
}

func NewProviderManager(myProvider *Provider) *ProviderManager {
	pm := &ProviderManager{
		MyProvider:   myProvider,
		Providers:    []*Provider{},
		ProvidersMap: make(map[Provider_Type][]*Provider),
	}
	return pm
}

func (pm *ProviderManager) AddProvider(p *Provider) {
	for _, pv := range pm.Providers {
		if pv.Id == p.Id {
			return
		}
	}
	mu.Lock()
	pm.Providers = append(pm.Providers, p)
	pm.CreateProvidersMap()
	mu.Unlock()
	//log.Printf("Providers: %v\n", pm.Providers)
}

func (pm *ProviderManager) SetProviders(ps []*Provider) {
	mu.Lock()
	pm.Providers = ps
	pm.CreateProvidersMap()
	mu.Unlock()
	//log.Printf("Providers: %v\n", pm.Providers)
}

func (pm *ProviderManager) GetProviders() []*Provider {
	return pm.Providers
}

func (pm *ProviderManager) DeleteProvider(id uint64) {
	newProviders := make([]*Provider, 0)
	for _, provider := range pm.Providers {
		if provider.Id == id {
			continue
		}
		newProviders = append(newProviders, provider)
	}
	pm.Providers = newProviders
	pm.CreateProvidersMap()
}

func (pm *ProviderManager) GetTargets(typeList []Provider_Type) []uint64 {
	idList := make([]uint64, 0)
	for _, tp := range typeList {
		for _, p := range pm.ProvidersMap[tp] {
			id := p.GetId()
			idList = append(idList, id)
		}
	}
	return idList
}

func (pm *ProviderManager) CreateProvidersMap() {
	providersMap := make(map[Provider_Type][]*Provider)

	for _, p := range pm.Providers {
		if p.GetId() != pm.MyProvider.GetId() { // 自分は含まない
			switch p.GetType() {
			case Provider_MASTER:
				providersMap[Provider_MASTER] = append(providersMap[Provider_MASTER], p)
			case Provider_WORKER:
				providersMap[Provider_WORKER] = append(providersMap[Provider_WORKER], p)
			case Provider_GATEWAY:
				providersMap[Provider_GATEWAY] = append(providersMap[Provider_GATEWAY], p)
			case Provider_VISUALIZATION:
				providersMap[Provider_VISUALIZATION] = append(providersMap[Provider_VISUALIZATION], p)
			case Provider_DATABASE:
				providersMap[Provider_DATABASE] = append(providersMap[Provider_DATABASE], p)
			case Provider_AGENT:
				providersMap[Provider_AGENT] = append(providersMap[Provider_AGENT], p)
			}
		}
	}
	pm.ProvidersMap = providersMap

}

////////////////////////////////////////////////////////////
////////////                   Provider API           ///////////
///////////////////////////////////////////////////////////
// Callbackと

type SclientOpt struct {
	Sclient      *sxutil.SXServiceClient
	ChType       uint32
	MBusCallback func(*sxutil.SXServiceClient, *sxapi.MbusMsg)
	ArgJson      string
	Providers    []*Provider
}

type ProviderAPI struct {
	Provider *Provider
	SimAPI *SimAPI
	ServerAddr string
	NodeAddr string
	SclientOpts map[uint32]*SclientOpt
}


func NewProviderAPI(provider *Provider, servAddr string, nodeAddr string, cb CallbackInterface ) *ProviderAPI {
	simapi := NewSimAPI(provider)
	api := &ProviderAPI{
		Provider: provider,
		SimAPI: simapi,
		ServerAddr: servAddr,
		NodeAddr: nodeAddr,
		SclientOpts: map[uint32]*SclientOpt{
			uint32(ChannelType_CLOCK): &SclientOpt{
				ChType:       uint32(ChannelType_CLOCK),
				MBusCallback: GetClockCallback(simapi, cb),
				ArgJson:      fmt.Sprintf("{Client:%s_Clock}", provider.Name),
			},
			uint32(ChannelType_PROVIDER): &SclientOpt{
				ChType:       uint32(ChannelType_PROVIDER),
				MBusCallback: GetProviderCallback(simapi, cb),
				ArgJson:      fmt.Sprintf("{Client:%s_Provider}", provider.Name),
			},
			uint32(ChannelType_AGENT): &SclientOpt{
				ChType:       uint32(ChannelType_AGENT),
				MBusCallback: GetAgentCallback(simapi, cb),
				ArgJson:      fmt.Sprintf("{Client:%s_Agent}", provider.Name),
			},
			uint32(ChannelType_AREA): &SclientOpt{
				ChType:       uint32(ChannelType_AREA),
				MBusCallback: GetAreaCallback(simapi, cb),
				ArgJson:      fmt.Sprintf("{Client:%s_Clock}", provider.Name),
			},
		},
	}
	return api
}

// Connect: Worker Nodeに接続する
func (ap *ProviderAPI) ConnectServer(flag bool) error {
	channelTypes := []uint32{}
	for _, opt := range ap.SclientOpts {
		channelTypes = append(channelTypes, opt.ChType)
	}
	// FIX
	ni := sxutil.GetDefaultNodeServInfo()
	if flag {
		ni = sxutil.NewNodeServInfo()
	}
	ap.SimAPI.RegisterNodeLoop(ni, ap.NodeAddr, ap.Provider.Name, channelTypes)

	// Register Synerex Server
	client := ap.SimAPI.RegisterSynerexLoop(ap.ServerAddr)

	// Register Callback
	ap.SimAPI.RegisterSXServiceClients(ni, client, ap.SclientOpts)
	return nil
}

func (ap *ProviderAPI) RegisterProvider() error {
	sclient := ap.SclientOpts[uint32(ChannelType_PROVIDER)].Sclient
	ap.SimAPI.RegisterProviderLoop(sclient, ap.SimAPI)
	return nil
}

func (ap *ProviderAPI) UpdateProviders(targets []uint64, providers []*Provider) error{
	sclient := ap.SclientOpts[uint32(ChannelType_PROVIDER)].Sclient
	_, err := ap.SimAPI.UpdateProvidersRequest(sclient, targets, providers)
	return err
}

func (ap *ProviderAPI) GetAgents(targets []uint64) ([]*Agent, error){
	sclient := ap.SclientOpts[uint32(ChannelType_AGENT)].Sclient
	agents := []*Agent{}
	simMsgs, err := ap.SimAPI.GetAgentRequest(sclient, targets)
	//log.Printf("simMsgs: %v %v", len(simMsgs), simMsgs)
	for _, simMsg := range simMsgs {
		agentsData := simMsg.GetGetAgentResponse().GetAgents()
		agents = append(agents, agentsData...)
	}
	return agents, err
}

func (ap *ProviderAPI) SetAgents(targets []uint64, agents []*Agent) error{
	sclient := ap.SclientOpts[uint32(ChannelType_AGENT)].Sclient
	_, err := ap.SimAPI.SetAgentRequest(sclient, targets, agents)
	return err
}

func (ap *ProviderAPI) ForwardClockInit(targets []uint64) error{
	sclient := ap.SclientOpts[uint32(ChannelType_CLOCK)].Sclient
	_, err := ap.SimAPI.ForwardClockInitRequest(sclient, targets)
	return err
}

func (ap *ProviderAPI) ForwardClockMain(targets []uint64) error{
	sclient := ap.SclientOpts[uint32(ChannelType_CLOCK)].Sclient
	_, err := ap.SimAPI.ForwardClockMainRequest(sclient, targets)
	return err
}

func (ap *ProviderAPI) ForwardClockTerminate(targets []uint64) error{
	sclient := ap.SclientOpts[uint32(ChannelType_CLOCK)].Sclient
	_, err := ap.SimAPI.ForwardClockTerminateRequest(sclient, targets)
	return err
}

func (ap *ProviderAPI) SetClock(targets []uint64, clock *Clock) error{
	sclient := ap.SclientOpts[uint32(ChannelType_CLOCK)].Sclient
	_, err := ap.SimAPI.SetClockRequest(sclient, targets, clock)
	return err
}

func (ap *ProviderAPI) SetArea(targets []uint64) {
	//sclient := ap.SclientOpts[uint32(ChannelType_AGENT)].Sclient
	//ap.SimAPI.SetAreaRequest(sclient, targets, clock)
}

func (ap *ProviderAPI) Reset(targets []uint64) error{
	sclient := ap.SclientOpts[uint32(ChannelType_PROVIDER)].Sclient
	_, err := ap.SimAPI.ResetRequest(sclient, targets)
	return err
}


func (ap *ProviderAPI) SimulatorRequest(targets []uint64, req *SimulatorRequest) error{
	sclient := ap.SclientOpts[uint32(ChannelType_PROVIDER)].Sclient
	_, err := ap.SimAPI.SimulatorRequest(sclient, targets, req)
	return err
}



////////////////////////////////////////////////////////////
////////////                   SimAPI           ///////////
///////////////////////////////////////////////////////////
// Synerexへのデータ送受信+Waiterによる待機処理

type SimAPI struct {
	Waiter   *Waiter
	Provider *Provider
}

func NewSimAPI(provider *Provider) *SimAPI {
	s := &SimAPI{
		Waiter:   NewWaiter(),
		Provider: provider,
	}
	return s
}

////////////////////////////////////////////////////////////
////////////        Supply Demand Function       ///////////
///////////////////////////////////////////////////////////

func (s *SimAPI) ResponseSimMsg(sclient *sxutil.SXServiceClient, simMsg *SimMsg, targetId uint64) error {

	cdata, _ := proto.Marshal(simMsg)
	msg := &sxapi.MbusMsg{
		TargetId: targetId,
		Cdata: &sxapi.Content{
			Entity: cdata,
		},
	}

	//mu.Lock()
	//log.Printf("responseSimMsg: %v", simMsg)
	ctx := context.Background()
	err := sclient.SendMbusMsg(ctx, msg)
	if err != nil {
		return err
	}
	//mu.Unlock()

	return nil
}

func (s *SimAPI) RequestSimMsg(sclient *sxutil.SXServiceClient, targets []uint64, simMsg *SimMsg) ([]*SimMsg, error) {

	cdata, _ := proto.Marshal(simMsg)
	msg := &sxapi.MbusMsg{
		Cdata: &sxapi.Content{
			Entity: cdata,
		},
	}
	simMsgId := simMsg.GetMsgId()
	bufSize := 10 // Channel Buffer Size
	s.Waiter.RegisterWaitCh(simMsgId, bufSize)

	mu.Lock()
	ctx := context.Background()
	err := sclient.SendMbusMsg(ctx, msg)
	if err != nil {
		return nil, err
	}
	mu.Unlock()

	// waitする
	simMsgs := []*SimMsg{}
	// if len(targets) = 0, wait until recieving first msg
	if len(targets) != 0 {
		simMsgs, err = s.Waiter.WaitMsg(simMsgId, targets, 1000)
		//log.Printf("debug: %v, %v", simMsgs, err)
		if err != nil {
			return nil, err
		}
	}

	return simMsgs, nil
}

func (s *SimAPI) SendMsgToWait(msg *sxapi.MbusMsg) {
	s.Waiter.SendMsgToWait(msg)
}

///////////////////////////////////////////
/////////////      Waiter      ////////////
//////////////////////////////////////////

type Waiter struct {
	WaitChMap map[uint64]chan *SimMsg
	MsgMap    map[uint64][]*SimMsg
}

func NewWaiter() *Waiter {
	w := &Waiter{
		WaitChMap: make(map[uint64]chan *SimMsg),
		MsgMap:    make(map[uint64][]*SimMsg),
	}
	return w
}

func (w *Waiter) RegisterWaitCh(simMsgId uint64, bufSize int) {
	mu.Lock()
	defer mu.Unlock()
	waitCh := make(chan *SimMsg, bufSize)
	w.WaitChMap[simMsgId] = waitCh
	w.MsgMap[simMsgId] = make([]*SimMsg, 0)
	//log.Printf("RegisterWaitCh\n")
}

func (w *Waiter) WaitMsg(simMsgId uint64, targets []uint64, timeout uint64) ([]*SimMsg, error) {

	mu.Lock()
	waitCh := w.WaitChMap[simMsgId]
	mu.Unlock()

	wg := sync.WaitGroup{}
	wg.Add(1)
	var err error
	//log.Printf("WaitMsg %v\n", simMsgId)
	go func() {
		for {
			select {
			case simMsg, _ := <-waitCh:
				//log.Printf("\nGetmsg\n", targets, simMsg.GetMsgId(), simMsgId)
				mu.Lock()
				// spのidがidListに入っているか
				if simMsg.GetMsgId() == simMsgId {
					w.MsgMap[simMsgId] = append(w.MsgMap[simMsgId], simMsg)
					//log.Printf("\nDone0\n")
					//log.Printf("GetSimMsg\n", targets, simMsg.GetMsgId(), simMsgId)
					// 同期が終了したかどうか
					if w.IsFinishWait(simMsgId, targets) {
						mu.Unlock()
						wg.Done()
						//log.Printf("Done\n")
						return
					}
				}
				mu.Unlock()
				//case <-time.After(time.Duration(timeout) * time.Millisecond):
				//	err = log.Errorf("Timeout Error")
				//	log.Printf("\nTimeout\n")
				//	wg.Done()
				//	return
			}
		}
	}()
	wg.Wait()
	mu.Lock()
	msgs := w.MsgMap[simMsgId]
	// delete key value
	delete(w.MsgMap, simMsgId)
	delete(w.WaitChMap, simMsgId)
	mu.Unlock()
	//log.Printf("\nsimmsgs %v %v\n", msgs, err)
	return msgs, err
}

func (w *Waiter) SendMsgToWait(msg *sxapi.MbusMsg) {
	simMsg := &SimMsg{}
	proto.Unmarshal(msg.Cdata.Entity, simMsg)
	mu.Lock()
	waitCh, ok := w.WaitChMap[simMsg.GetMsgId()]
	mu.Unlock()
	if ok {
		waitCh <- simMsg
	}
	//mu.Unlock()
	//log.Printf("\nSendMsgTowait\n")
}

func (w *Waiter) IsFinishWait(simMsgId uint64, targets []uint64) bool {
	//log.Printf("\nIsFinishWait\n", targets, simMsgId)
	//mu.Lock()
	//defer mu.Unlock()
	for _, target := range targets {
		//log.Printf("\nIsFini2\n")
		targetId := target
		isExist := false
		for _, simMsg := range w.MsgMap[simMsgId] {
			if targetId == simMsg.GetSenderId() || targetId == 0 {
				isExist = true
			}
		}
		if isExist == false {
			return false
		}
	}
	//mu.Unlock()
	//log.Printf("\nIsFinishWait2\n", targets, simMsgId)
	return true
}

///////////////////////////////////////////
/////////////    Area API   //////////////
//////////////////////////////////////////

// Areaを送るDemand
func (s *SimAPI) SendAreaInfoRequest(sclient *sxutil.SXServiceClient, targets []uint64, areas []*Area) ([]*SimMsg, error) {
	if len(targets) == 0 {
		return []*SimMsg{}, nil
	}
	uid, _ := uuid.NewRandom()
	sendAreaInfoRequest := &SendAreaInfoRequest{
		Areas: areas,
	}

	msgId := uint64(uid.ID())
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_SEND_AREA_INFO_REQUEST,
		Data:     &SimMsg_SendAreaInfoRequest{sendAreaInfoRequest},
	}

	sps, err := s.RequestSimMsg(sclient, targets, simMsg)

	return sps, err
}

// Agentのセット完了
func (s *SimAPI) SendAreaInfoResponse(sclient *sxutil.SXServiceClient, msgId uint64, targetId uint64) uint64 {
	sendAreaInfoResponse := &SendAreaInfoResponse{}

	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_SEND_AREA_INFO_RESPONSE,
		Data:     &SimMsg_SendAreaInfoResponse{sendAreaInfoResponse},
	}

	s.ResponseSimMsg(sclient, simMsg, targetId)

	return msgId
}

///////////////////////////////////////////
/////////////   Agent API   //////////////
//////////////////////////////////////////

// AgentをセットするDemand
func (s *SimAPI) SetAgentRequest(sclient *sxutil.SXServiceClient, targets []uint64, agents []*Agent) ([]*SimMsg, error) {
	if len(targets) == 0 {
		return []*SimMsg{}, nil
	}

	uid, _ := uuid.NewRandom()
	setAgentRequest := &SetAgentRequest{
		Agents: agents,
	}

	msgId := uint64(uid.ID())
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_SET_AGENT_REQUEST,
		Data:     &SimMsg_SetAgentRequest{setAgentRequest},
	}

	sps, err := s.RequestSimMsg(sclient, targets, simMsg)

	return sps, err
}

// Agentのセット完了
func (s *SimAPI) SetAgentResponse(sclient *sxutil.SXServiceClient, msgId uint64, targetId uint64) uint64 {
	setAgentResponse := &SetAgentResponse{}

	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_SET_AGENT_RESPONSE,
		Data:     &SimMsg_SetAgentResponse{setAgentResponse},
	}

	s.ResponseSimMsg(sclient, simMsg, targetId)

	return msgId
}

// AgentをセットするDemand
func (s *SimAPI) GetAgentRequest(sclient *sxutil.SXServiceClient, targets []uint64) ([]*SimMsg, error) {
	if len(targets) == 0 {
		return []*SimMsg{}, nil
	}

	uid, _ := uuid.NewRandom()
	getAgentRequest := &GetAgentRequest{}

	msgId := uint64(uid.ID())
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_GET_AGENT_REQUEST,
		Data:     &SimMsg_GetAgentRequest{getAgentRequest},
	}

	sps, err := s.RequestSimMsg(sclient, targets, simMsg)

	return sps, err
}

// Agentのセット完了
func (s *SimAPI) GetAgentResponse(sclient *sxutil.SXServiceClient, msgId uint64, targetId uint64, agents []*Agent) uint64 {
	getAgentResponse := &GetAgentResponse{
		Agents: agents,
	}

	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_GET_AGENT_RESPONSE,
		Data:     &SimMsg_GetAgentResponse{getAgentResponse},
	}

	s.ResponseSimMsg(sclient, simMsg, targetId)

	return msgId
}

///////////////////////////////////////////
////  Provider API Reset and Sim Request  /
//////////////////////////////////////////

// リセットするDemand
func (s *SimAPI) ResetRequest(sclient *sxutil.SXServiceClient, targets []uint64) ([]*SimMsg, error) {
	if len(targets) == 0 {
		return []*SimMsg{}, nil
	}
	resetRequest := &ResetRequest{}
	uid, _ := uuid.NewRandom()
	msgId := uint64(uid.ID())
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_RESET_REQUEST,
		Data:     &SimMsg_ResetRequest{resetRequest},
	}

	sps, err := s.RequestSimMsg(sclient, targets, simMsg)

	return sps, err
}

// リセットするSupply
func (s *SimAPI) ResetResponse(sclient *sxutil.SXServiceClient, msgId uint64, targetId uint64) uint64 {
	resetResponse := &ResetResponse{}

	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_RESET_RESPONSE,
		Data:     &SimMsg_ResetResponse{resetResponse},
	}

	s.ResponseSimMsg(sclient, simMsg, targetId)

	return msgId
}

// Simulator独自のデータをやりとりするDemand
func (s *SimAPI) SimulatorRequest(sclient *sxutil.SXServiceClient, targets []uint64, simReq *SimulatorRequest) ([]*SimMsg, error) {
	if len(targets) == 0 {
		return []*SimMsg{}, nil
	}
	uid, _ := uuid.NewRandom()
	msgId := uint64(uid.ID())
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_SIMULATOR_REQUEST,
		Data:     &SimMsg_SimulatorRequest{simReq},
	}

	sps, err := s.RequestSimMsg(sclient, targets, simMsg)

	return sps, err
}

// Simulator独自のデータをやりとりするSupply
func (s *SimAPI) SimulatorResponse(sclient *sxutil.SXServiceClient, msgId uint64, targetId uint64) uint64 {
	simulatorResponse := &SimulatorResponse{}

	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_SIMULATOR_RESPONSE,
		Data:     &SimMsg_SimulatorResponse{simulatorResponse},
	}

	s.ResponseSimMsg(sclient, simMsg, targetId)

	return msgId
}

///////////////////////////////////////////
/////////////   Provider API   //////////////
//////////////////////////////////////////

// Providerを登録するDemand
func (s *SimAPI) RegisterProviderRequest(sclient *sxutil.SXServiceClient, targets []uint64, provider *Provider) ([]*SimMsg, error) {
	if len(targets) == 0 {
		return []*SimMsg{}, nil
	}
	registerProviderRequest := &RegisterProviderRequest{
		Provider: provider,
	}

	log.Printf("provider %v %v\n", provider)
	uid, _ := uuid.NewRandom()
	msgId := uint64(uid.ID())
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_REGISTER_PROVIDER_REQUEST,
		Data:     &SimMsg_RegisterProviderRequest{registerProviderRequest},
	}

	sps, err := s.RequestSimMsg(sclient, targets, simMsg)
	//log.Printf("\nsps %v %v\n", sps, err)

	return sps, err
}

// Providerを登録するSupply
func (s *SimAPI) RegisterProviderResponse(sclient *sxutil.SXServiceClient, msgId uint64, targetId uint64, providerInfo *Provider) uint64 {
	registerProviderResponse := &RegisterProviderResponse{
		Provider: providerInfo,
	}

	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_REGISTER_PROVIDER_RESPONSE,
		Data:     &SimMsg_RegisterProviderResponse{registerProviderResponse},
	}

	s.ResponseSimMsg(sclient, simMsg, targetId)

	return msgId
}

// Providerを登録するDemand
func (s *SimAPI) UpdateProvidersRequest(sclient *sxutil.SXServiceClient, targets []uint64, providers []*Provider) ([]*SimMsg, error) {
	if len(targets) == 0 {
		return []*SimMsg{}, nil
	}
	updateProvidersRequest := &UpdateProvidersRequest{
		Providers: providers,
	}

	uid, _ := uuid.NewRandom()
	msgId := uint64(uid.ID())
	//log.Printf("msgId: %v", msgId)
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_UPDATE_PROVIDERS_REQUEST,
		Data:     &SimMsg_UpdateProvidersRequest{updateProvidersRequest},
	}

	sps, err := s.RequestSimMsg(sclient, targets, simMsg)

	return sps, err
}

// Providerを登録するSupply
func (s *SimAPI) UpdateProvidersResponse(sclient *sxutil.SXServiceClient, msgId uint64, targetId uint64) uint64 {
	updateProvidersResponse := &UpdateProvidersResponse{}

	//log.Printf("response simMsg\n")
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_UPDATE_PROVIDERS_RESPONSE,
		Data:     &SimMsg_UpdateProvidersResponse{updateProvidersResponse},
	}

	//log.Printf("response simMsg: %v\n", simMsg)
	s.ResponseSimMsg(sclient, simMsg, targetId)

	return msgId
}

///////////////////////////////////////////
/////////////   Clock API   //////////////
//////////////////////////////////////////

func (s *SimAPI) SetClockRequest(sclient *sxutil.SXServiceClient, targets []uint64, clockInfo *Clock) ([]*SimMsg, error) {
	if len(targets) == 0 {
		return []*SimMsg{}, nil
	}
	setClockRequest := &SetClockRequest{
		Clock: clockInfo,
	}

	uid, _ := uuid.NewRandom()
	msgId := uint64(uid.ID())
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_SET_CLOCK_REQUEST,
		Data:     &SimMsg_SetClockRequest{setClockRequest},
	}

	sps, err := s.RequestSimMsg(sclient, targets, simMsg)

	return sps, err
}

// Agentを取得するSupply
func (s *SimAPI) SetClockResponse(sclient *sxutil.SXServiceClient, msgId uint64, targetId uint64) uint64 {
	setClockResponse := &SetClockResponse{}

	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_SET_CLOCK_RESPONSE,
		Data:     &SimMsg_SetClockResponse{setClockResponse},
	}

	s.ResponseSimMsg(sclient, simMsg, targetId)

	return msgId
}

func (s *SimAPI) ForwardClockRequest(sclient *sxutil.SXServiceClient, targets []uint64) ([]*SimMsg, error) {
	if len(targets) == 0 {
		return []*SimMsg{}, nil
	}
	forwardClockRequest := &ForwardClockRequest{}

	uid, _ := uuid.NewRandom()
	msgId := uint64(uid.ID())
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_FORWARD_CLOCK_REQUEST,
		Data:     &SimMsg_ForwardClockRequest{forwardClockRequest},
	}

	sps, err := s.RequestSimMsg(sclient, targets, simMsg)

	return sps, err
}

// Agentを取得するSupply
func (s *SimAPI) ForwardClockResponse(sclient *sxutil.SXServiceClient, msgId uint64, targetId uint64) uint64 {
	forwardClockResponse := &ForwardClockResponse{}

	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_FORWARD_CLOCK_RESPONSE,
		Data:     &SimMsg_ForwardClockResponse{forwardClockResponse},
	}

	s.ResponseSimMsg(sclient, simMsg, targetId)

	return msgId
}

func (s *SimAPI) ForwardClockInitRequest(sclient *sxutil.SXServiceClient, targets []uint64) ([]*SimMsg, error) {
	if len(targets) == 0 {
		return []*SimMsg{}, nil
	}
	forwardClockInitRequest := &ForwardClockInitRequest{}

	uid, _ := uuid.NewRandom()
	msgId := uint64(uid.ID())
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_FORWARD_CLOCK_INIT_REQUEST,
		Data:     &SimMsg_ForwardClockInitRequest{forwardClockInitRequest},
	}

	sps, err := s.RequestSimMsg(sclient, targets, simMsg)

	return sps, err
}

// Agentを取得するSupply
func (s *SimAPI) ForwardClockInitResponse(sclient *sxutil.SXServiceClient, msgId uint64, targetId uint64) uint64 {
	forwardClockInitResponse := &ForwardClockInitResponse{}

	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_FORWARD_CLOCK_INIT_RESPONSE,
		Data:     &SimMsg_ForwardClockInitResponse{forwardClockInitResponse},
	}

	s.ResponseSimMsg(sclient, simMsg, targetId)

	return msgId
}

func (s *SimAPI) ForwardClockMainRequest(sclient *sxutil.SXServiceClient, targets []uint64) ([]*SimMsg, error) {
	if len(targets) == 0 {
		return []*SimMsg{}, nil
	}
	forwardClockMainRequest := &ForwardClockMainRequest{}

	uid, _ := uuid.NewRandom()
	msgId := uint64(uid.ID())
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_FORWARD_CLOCK_MAIN_REQUEST,
		Data:     &SimMsg_ForwardClockMainRequest{forwardClockMainRequest},
	}

	sps, err := s.RequestSimMsg(sclient, targets, simMsg)

	return sps, err
}

// Agentを取得するSupply
func (s *SimAPI) ForwardClockMainResponse(sclient *sxutil.SXServiceClient, msgId uint64, targetId uint64) uint64 {
	forwardClockMainResponse := &ForwardClockMainResponse{}

	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_FORWARD_CLOCK_MAIN_RESPONSE,
		Data:     &SimMsg_ForwardClockMainResponse{forwardClockMainResponse},
	}

	s.ResponseSimMsg(sclient, simMsg, targetId)

	return msgId
}

func (s *SimAPI) ForwardClockTerminateRequest(sclient *sxutil.SXServiceClient, targets []uint64) ([]*SimMsg, error) {
	if len(targets) == 0 {
		return []*SimMsg{}, nil
	}
	forwardClockTerminateRequest := &ForwardClockTerminateRequest{}

	uid, _ := uuid.NewRandom()
	msgId := uint64(uid.ID())
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_FORWARD_CLOCK_TERMINATE_REQUEST,
		Data:     &SimMsg_ForwardClockTerminateRequest{forwardClockTerminateRequest},
	}

	sps, err := s.RequestSimMsg(sclient, targets, simMsg)

	return sps, err
}

// Agentを取得するSupply
func (s *SimAPI) ForwardClockTerminateResponse(sclient *sxutil.SXServiceClient, msgId uint64, targetId uint64) uint64 {
	forwardClockTerminateResponse := &ForwardClockTerminateResponse{}

	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_FORWARD_CLOCK_TERMINATE_RESPONSE,
		Data:     &SimMsg_ForwardClockTerminateResponse{forwardClockTerminateResponse},
	}

	s.ResponseSimMsg(sclient, simMsg, targetId)

	return msgId
}

func (s *SimAPI) StartClockRequest(sclient *sxutil.SXServiceClient, targets []uint64) ([]*SimMsg, error) {
	if len(targets) == 0 {
		return []*SimMsg{}, nil
	}
	startClockRequest := &StartClockRequest{}

	uid, _ := uuid.NewRandom()
	msgId := uint64(uid.ID())
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_START_CLOCK_REQUEST,
		Data:     &SimMsg_StartClockRequest{startClockRequest},
	}

	sps, err := s.RequestSimMsg(sclient, targets, simMsg)

	return sps, err
}

// Agentを取得するSupply
func (s *SimAPI) StartClockResponse(sclient *sxutil.SXServiceClient, msgId uint64, targetId uint64) uint64 {
	startClockResponse := &StartClockResponse{}

	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_START_CLOCK_RESPONSE,
		Data:     &SimMsg_StartClockResponse{startClockResponse},
	}

	s.ResponseSimMsg(sclient, simMsg, targetId)

	return msgId
}

func (s *SimAPI) StopClockRequest(sclient *sxutil.SXServiceClient, targets []uint64) ([]*SimMsg, error) {
	if len(targets) == 0 {
		return []*SimMsg{}, nil
	}
	stopClockRequest := &StopClockRequest{}

	uid, _ := uuid.NewRandom()
	msgId := uint64(uid.ID())
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_STOP_CLOCK_REQUEST,
		Data:     &SimMsg_StopClockRequest{stopClockRequest},
	}

	sps, err := s.RequestSimMsg(sclient, targets, simMsg)

	return sps, err
}

// Agentを取得するSupply
func (s *SimAPI) StopClockResponse(sclient *sxutil.SXServiceClient, msgId uint64, targetId uint64) uint64 {
	stopClockResponse := &StopClockResponse{}

	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_STOP_CLOCK_RESPONSE,
		Data:     &SimMsg_StopClockResponse{stopClockResponse},
	}

	s.ResponseSimMsg(sclient, simMsg, targetId)

	return msgId
}

// NodeServに繋がるまで繰り返す
func (s *SimAPI) RegisterNodeLoop(ni *sxutil.NodeServInfo, nodesrv string, name string, chTypes []uint32) *sxutil.NodeServInfo {
	go sxutil.HandleSigInt() // Ctl+cを認識させる
	for {
		_, err := ni.RegisterNodeWithCmd(nodesrv, name, chTypes, nil, nil)
		if err != nil {
			time.Sleep(1 * time.Second)
		} else {
			sxutil.RegisterDeferFunction(sxutil.UnRegisterNode)
			//ni := sxutil.GetDefaultNodeServInfo()
			return ni
		}
	}
}

func (s *SimAPI) RegisterSXServiceClients(ni *sxutil.NodeServInfo, client sxapi.SynerexClient, opts map[uint32]*SclientOpt) map[uint32]*SclientOpt {
	for key, opt := range opts {
		sclient := ni.NewSXServiceClient(client, opt.ChType, opt.ArgJson) // service client
		sclient.MbusID = sxutil.IDType(opt.ChType)                        // MbusIDをChTypeに変更
		//log.Printf("debug MbusID: %d", sclient.MbusID)
		opts[key].Sclient = sclient
		go s.SubscribeMbusLoop(sclient, opt.MBusCallback)
	}
	return opts
}

func (s *SimAPI) SubscribeMbusLoop(sclient *sxutil.SXServiceClient, mbcb func(*sxutil.SXServiceClient, *sxapi.MbusMsg)) {
	//called as goroutine
	ctx := context.Background() // should check proper context
	sxutil.RegisterDeferFunction(func() {
		sclient.CloseMbus(ctx)
	})
	for {
		sclient.SubscribeMbus(ctx, mbcb)
		// comes here if channel closed
		time.Sleep(1 * time.Second)
	}
}

// Synerexに繋がるまで繰り返す
func (s *SimAPI) RegisterSynerexLoop(sxServerAddress string) sxapi.SynerexClient {
	for {
		client := sxutil.GrpcConnectServer(sxServerAddress)
		if client == nil {
			time.Sleep(1 * time.Second)
		} else {
			return client
		}
	}
}

// WorkerやMasterにProviderを登録する
func (s *SimAPI) RegisterProviderLoop(sclient *sxutil.SXServiceClient, simapi *SimAPI) *Provider {
	// masterへ登録
	targets := []uint64{0}
	//bc.simapi.RegisterProviderRequest(sclient, targets, bc.simapi.Provider)
	var provider *Provider
	ch := make(chan struct{})
	go func() {
		for {
			log.Printf("RegistProviderRequst %v", simapi.Provider.Id)
			msgs, err := simapi.RegisterProviderRequest(sclient, targets, simapi.Provider)
			if err != nil {
				time.Sleep(2 * time.Second)

			} else {
				provider = msgs[0].GetRegisterProviderResponse().GetProvider()
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

func GetAgentCallback(simapi *SimAPI, callback CallbackInterface) func(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	newCb := NewBaseCallback(simapi, callback)
	return newCb.AgentCallback
}

func GetProviderCallback(simapi *SimAPI, callback CallbackInterface) func(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	newCb := NewBaseCallback(simapi, callback)
	return newCb.ProviderCallback
}

func GetClockCallback(simapi *SimAPI, callback CallbackInterface) func(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	newCb := NewBaseCallback(simapi, callback)
	return newCb.ClockCallback
}

func GetAreaCallback(simapi *SimAPI, callback CallbackInterface) func(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	newCb := NewBaseCallback(simapi, callback)
	return newCb.ProviderCallback
}

type CallbackInterface interface {
	SetAgentRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	SetAgentResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	GetAgentRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg) []*Agent
	GetAgentResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	RegisterProviderRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg) *Provider
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
	ResetRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	ResetResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	SimulatorRequest(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
	SimulatorResponse(clt *sxutil.SXServiceClient, simMsg *sxapi.MbusMsg)
}

type BaseCallback struct {
	simapi *SimAPI
	CallbackInterface
}

func NewBaseCallback(simapi *SimAPI, cbif CallbackInterface) *BaseCallback {
	bc := &BaseCallback{
		CallbackInterface: cbif,
		simapi:            simapi,
	}
	return bc
}

func (bc *BaseCallback) AgentCallback(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	go func() {
		simMsg := &SimMsg{}
		proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
		//targets := []uint64{simMsg.GetSenderId()}
		//log.Printf("get Agent Callback %v\n", simMsg)
		msgId := simMsg.GetMsgId()
		switch simMsg.GetType() {
		case MsgType_SET_AGENT_REQUEST:
			bc.SetAgentRequest(clt, msg)
			// response
			targetId := msg.GetSenderId()
			bc.simapi.SetAgentResponse(clt, msgId, targetId)
		case MsgType_GET_AGENT_REQUEST:

			agents := bc.GetAgentRequest(clt, msg)
			// response
			targetId := msg.GetSenderId()
			bc.simapi.GetAgentResponse(clt, msgId, targetId, agents)

		case MsgType_SET_AGENT_RESPONSE:
			bc.SetAgentResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)
		case MsgType_GET_AGENT_RESPONSE:

			bc.GetAgentResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)

		}
		return
	}()
}

func (bc *BaseCallback) ProviderCallback(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	go func() {
		simMsg := &SimMsg{}
		proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
		//targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()

		//log.Printf("get Provider Callback %v\n", simMsg)
		switch simMsg.GetType() {
		case MsgType_REGISTER_PROVIDER_REQUEST:

			provider := bc.RegisterProviderRequest(clt, msg)
			// response
			targetId := msg.GetSenderId()
			bc.simapi.RegisterProviderResponse(clt, msgId, targetId, provider)

		case MsgType_UPDATE_PROVIDERS_REQUEST:

			bc.UpdateProvidersRequest(clt, msg)
			// response
			targetId := msg.GetSenderId()
			bc.simapi.UpdateProvidersResponse(clt, msgId, targetId)

		case MsgType_REGISTER_PROVIDER_RESPONSE:

			bc.RegisterProviderResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)

		case MsgType_UPDATE_PROVIDERS_RESPONSE:

			bc.UpdateProvidersResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)


		case MsgType_RESET_REQUEST:
			bc.ResetRequest(clt, msg)
			// response
			targetId := msg.GetSenderId()
			bc.simapi.ResetResponse(clt, msgId, targetId)

		case MsgType_RESET_RESPONSE:
			bc.ResetResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)
		
		case MsgType_SIMULATOR_REQUEST:
			bc.SimulatorRequest(clt, msg)
			// response
			targetId := msg.GetSenderId()
			bc.simapi.SimulatorResponse(clt, msgId, targetId)

		case MsgType_SIMULATOR_RESPONSE:
			bc.SimulatorResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)
		}
		return
	}()
}

func (bc *BaseCallback) ClockCallback(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	go func() {
		simMsg := &SimMsg{}
		proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
		//targets := []uint64{simMsg.GetSenderId()}
		//log.Printf("get Clock Callback %v\n", simMsg)
		msgId := simMsg.GetMsgId()
		switch simMsg.GetType() {
		case MsgType_SET_CLOCK_REQUEST:
			bc.SetClockRequest(clt, msg)
			// response
			targetId := msg.GetSenderId()
			bc.simapi.SetClockResponse(clt, msgId, targetId)
		case MsgType_STOP_CLOCK_REQUEST:
			bc.StopClockRequest(clt, msg)
			// response
			targetId := msg.GetSenderId()
			bc.simapi.StopClockResponse(clt, msgId, targetId)
		case MsgType_START_CLOCK_REQUEST:
			bc.StartClockRequest(clt, msg)
			// response
			targetId := msg.GetSenderId()
			bc.simapi.StartClockResponse(clt, msgId, targetId)
		case MsgType_FORWARD_CLOCK_REQUEST:
			bc.ForwardClockRequest(clt, msg)
			// response
			targetId := msg.GetSenderId()
			bc.simapi.ForwardClockResponse(clt, msgId, targetId)
		case MsgType_FORWARD_CLOCK_INIT_REQUEST:
			bc.ForwardClockInitRequest(clt, msg)
			// response
			targetId := msg.GetSenderId()
			bc.simapi.ForwardClockInitResponse(clt, msgId, targetId)
		case MsgType_FORWARD_CLOCK_MAIN_REQUEST:
			bc.ForwardClockMainRequest(clt, msg)
			// response
			targetId := msg.GetSenderId()
			bc.simapi.ForwardClockMainResponse(clt, msgId, targetId)
		case MsgType_FORWARD_CLOCK_TERMINATE_REQUEST:
			bc.ForwardClockTerminateRequest(clt, msg)
			// response
			targetId := msg.GetSenderId()
			bc.simapi.ForwardClockTerminateResponse(clt, msgId, targetId)
		case MsgType_SET_CLOCK_RESPONSE:
			bc.SetClockResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)
		case MsgType_START_CLOCK_RESPONSE:
			bc.StartClockResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)
		case MsgType_STOP_CLOCK_RESPONSE:
			bc.StopClockResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)
		case MsgType_FORWARD_CLOCK_RESPONSE:
			bc.ForwardClockResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)
		case MsgType_FORWARD_CLOCK_INIT_RESPONSE:
			bc.ForwardClockInitResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)
		case MsgType_FORWARD_CLOCK_MAIN_RESPONSE:
			bc.ForwardClockMainResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)
		case MsgType_FORWARD_CLOCK_TERMINATE_RESPONSE:
			bc.ForwardClockTerminateResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)
		}
		return
	}()
}

func (bc *BaseCallback) AreaCallback(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	go func() {
		simMsg := &SimMsg{}
		//targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		//log.Printf("get Area Callback %v\n", simMsg)
		proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
		switch simMsg.GetType() {
		case MsgType_SEND_AREA_INFO_REQUEST:
			bc.SendAreaInfoRequest(clt, msg)
			// response
			targetId := msg.GetSenderId()
			bc.simapi.SendAreaInfoResponse(clt, msgId, targetId)
		case MsgType_SEND_AREA_INFO_RESPONSE:
			bc.SendAreaInfoResponse(clt, msg)
			bc.simapi.SendMsgToWait(msg)
		}
		return
	}()
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
func (cb Callback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*Agent {
	return []*Agent{}
}
func (cb Callback) GetAgentResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {}

// Provider
func (cb Callback) RegisterProviderRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) *Provider {
	return &Provider{}
}
func (cb Callback) RegisterProviderResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {}
func (cb Callback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)   {}
func (cb Callback) UpdateProvidersResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)  {}
func (cb Callback) ResetRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)   {}
func (cb Callback) ResetResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)  {}
func (cb Callback) SimulatorRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)   {}
func (cb Callback) SimulatorResponse(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg)  {}

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
