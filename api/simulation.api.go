package api

import (
	"context"
	"log"

	//"log"
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

// Waitする基準となるFilter　nilにすれば対象でなくなる
type Filter struct {
	TargetId uint64
}

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

func (s *SimAPI) RequestSimMsg(sclient *sxutil.SXServiceClient, filters []*Filter, simMsg *SimMsg) ([]*SimMsg, error) {

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
	// if len(filters) = 0, wait until recieving first msg
	if len(filters) != 0 {
		simMsgs, err = s.Waiter.WaitMsg(simMsgId, filters, 1000)
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

func (w *Waiter) WaitMsg(simMsgId uint64, filters []*Filter, timeout uint64) ([]*SimMsg, error) {

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
				//log.Printf("\nGetmsg\n", filters, simMsg.GetMsgId(), simMsgId)
				mu.Lock()
				// spのidがidListに入っているか
				if simMsg.GetMsgId() == simMsgId {
					w.MsgMap[simMsgId] = append(w.MsgMap[simMsgId], simMsg)
					//log.Printf("\nDone0\n")
					log.Printf("GetSimMsg\n", filters, simMsg.GetMsgId(), simMsgId)
					// 同期が終了したかどうか
					if w.IsFinishWait(simMsgId, filters) {
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
	//mu.Lock()
	simMsg := &SimMsg{}
	proto.Unmarshal(msg.Cdata.Entity, simMsg)
	waitCh, ok := w.WaitChMap[simMsg.GetMsgId()]
	if ok {
		waitCh <- simMsg
	}
	//mu.Unlock()
	//log.Printf("\nSendMsgTowait\n")
}

func (w *Waiter) IsFinishWait(simMsgId uint64, filters []*Filter) bool {
	//log.Printf("\nIsFinishWait\n", filters, simMsgId)
	//mu.Lock()
	//defer mu.Unlock()
	for _, filter := range filters {
		//log.Printf("\nIsFini2\n")
		targetId := filter.TargetId
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
func (s *SimAPI) SendAreaInfoRequest(sclient *sxutil.SXServiceClient, filters []*Filter, areas []*Area) ([]*SimMsg, error) {
	if len(filters) == 0 {
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

	sps, err := s.RequestSimMsg(sclient, filters, simMsg)

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
func (s *SimAPI) SetAgentRequest(sclient *sxutil.SXServiceClient, filters []*Filter, agents []*Agent) ([]*SimMsg, error) {
	if len(filters) == 0 {
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

	sps, err := s.RequestSimMsg(sclient, filters, simMsg)

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
func (s *SimAPI) GetAgentRequest(sclient *sxutil.SXServiceClient, filters []*Filter) ([]*SimMsg, error) {
	if len(filters) == 0 {
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

	sps, err := s.RequestSimMsg(sclient, filters, simMsg)

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
/////////////   Provider API   //////////////
//////////////////////////////////////////

// Providerを登録するDemand
func (s *SimAPI) RegisterProviderRequest(sclient *sxutil.SXServiceClient, filters []*Filter, providerInfo *Provider) ([]*SimMsg, error) {
	if len(filters) == 0 {
		return []*SimMsg{}, nil
	}
	registerProviderRequest := &RegisterProviderRequest{
		Provider: providerInfo,
	}

	uid, _ := uuid.NewRandom()
	msgId := uint64(uid.ID())
	simMsg := &SimMsg{
		MsgId:    msgId,
		SenderId: s.Provider.Id,
		Type:     MsgType_REGISTER_PROVIDER_REQUEST,
		Data:     &SimMsg_RegisterProviderRequest{registerProviderRequest},
	}

	sps, err := s.RequestSimMsg(sclient, filters, simMsg)
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
func (s *SimAPI) UpdateProvidersRequest(sclient *sxutil.SXServiceClient, filters []*Filter, providers []*Provider) ([]*SimMsg, error) {
	if len(filters) == 0 {
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

	sps, err := s.RequestSimMsg(sclient, filters, simMsg)

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

func (s *SimAPI) SetClockRequest(sclient *sxutil.SXServiceClient, filters []*Filter, clockInfo *Clock) ([]*SimMsg, error) {
	if len(filters) == 0 {
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

	sps, err := s.RequestSimMsg(sclient, filters, simMsg)

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

func (s *SimAPI) ForwardClockRequest(sclient *sxutil.SXServiceClient, filters []*Filter) ([]*SimMsg, error) {
	if len(filters) == 0 {
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

	sps, err := s.RequestSimMsg(sclient, filters, simMsg)

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

func (s *SimAPI) ForwardClockInitRequest(sclient *sxutil.SXServiceClient, filters []*Filter) ([]*SimMsg, error) {
	if len(filters) == 0 {
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

	sps, err := s.RequestSimMsg(sclient, filters, simMsg)

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

func (s *SimAPI) ForwardClockMainRequest(sclient *sxutil.SXServiceClient, filters []*Filter) ([]*SimMsg, error) {
	if len(filters) == 0 {
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

	sps, err := s.RequestSimMsg(sclient, filters, simMsg)

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

func (s *SimAPI) ForwardClockTerminateRequest(sclient *sxutil.SXServiceClient, filters []*Filter) ([]*SimMsg, error) {
	if len(filters) == 0 {
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

	sps, err := s.RequestSimMsg(sclient, filters, simMsg)

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

func (s *SimAPI) StartClockRequest(sclient *sxutil.SXServiceClient, filters []*Filter) ([]*SimMsg, error) {
	if len(filters) == 0 {
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

	sps, err := s.RequestSimMsg(sclient, filters, simMsg)

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

func (s *SimAPI) StopClockRequest(sclient *sxutil.SXServiceClient, filters []*Filter) ([]*SimMsg, error) {
	if len(filters) == 0 {
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

	sps, err := s.RequestSimMsg(sclient, filters, simMsg)

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
