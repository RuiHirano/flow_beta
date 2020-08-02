package main

import (
	"encoding/json"
	"flag"
	"log"

	//"math/rand"
	"time"

	//"strings"
	"sync"

	//"github.com/golang/protobuf/jsonpb"
	"runtime"

	api "github.com/RuiHirano/flow_beta/api"
	util "github.com/RuiHirano/flow_beta/util"
	"github.com/golang/protobuf/proto"
	sxapi "github.com/synerex/synerex_api"
	sxutil "github.com/synerex/synerex_sxutil"

	"fmt"
	"net/http"
	"os"

	"path/filepath"

	"github.com/google/uuid"
	gosocketio "github.com/mtfelian/golang-socketio"
)

var (
	myProvider     *api.Provider
	masterProvider *api.Provider
	pm             *util.ProviderManager
	mu             sync.Mutex
	assetsDir      http.FileSystem
	ioserv         *gosocketio.Server
	logger         *util.Logger
	agentsMessage  *Message

	sclientOptsMaster map[uint32]*util.SclientOpt
	sclientOptsVis    map[uint32]*util.SclientOpt
	simapi            *api.SimAPI
	servaddr          = flag.String("servaddr", getServerAddress(), "The Synerex Server Listening Address")
	nodeaddr          = flag.String("nodeaddr", getNodeservAddress(), "Node ID Server Address")
	masterServaddr    = flag.String("masterServaddr", getMasterServerAddress(), "Master Synerex Server Listening Address")
	masterNodeaddr    = flag.String("masterNodeaddr", getMasterNodeservAddress(), "Master Node ID Server Address")
	monitoraddr       = flag.String("monitoraddr", getMonitorAddress(), "Monitor Listening Address")
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

func getMonitorAddress() string {
	env := os.Getenv("MONITOR_ADDRESS")
	if env != "" {
		return env
	} else {
		return "127.0.0.1:9500"
	}
}

func getProviderName() string {
	env := os.Getenv("PROVIDER_NAME")
	if env != "" {
		return env
	} else {
		return "VisualizationProvider"
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

	vicb := &VisCallback{cb} // override
	sclientOptsVis = map[uint32]*util.SclientOpt{
		uint32(api.ChannelType_CLOCK): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_CLOCK),
			MBusCallback: util.GetClockCallback(simapi, vicb),
			ArgJson:      fmt.Sprintf("{Client:VisProvider_Clock}"),
		},
		uint32(api.ChannelType_PROVIDER): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_PROVIDER),
			MBusCallback: util.GetProviderCallback(simapi, vicb),
			ArgJson:      fmt.Sprintf("{Client:VisProvider_Provider}"),
		},
		uint32(api.ChannelType_AGENT): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AGENT),
			MBusCallback: util.GetAgentCallback(simapi, vicb),
			ArgJson:      fmt.Sprintf("{Client:VisProvider_Agent}"),
		},
		uint32(api.ChannelType_AREA): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AREA),
			MBusCallback: util.GetAreaCallback(simapi, vicb),
			ArgJson:      fmt.Sprintf("{Client:VisProvider_Area}"),
		},
	}

	logger = util.NewLogger()

}

////////////////////////////////////////////////////////////
////////////            Message Class           ///////////
///////////////////////////////////////////////////////////

type Message struct {
	ready     chan struct{}
	agents    []*api.Agent
	senderIds []uint64
	targets   []uint64
}

func NewMessage(targets []uint64) *Message {
	return &Message{ready: make(chan struct{}), agents: make([]*api.Agent, 0), senderIds: []uint64{}, targets: targets}
}

func (m *Message) IsFinish() bool {
	for _, tgt := range m.targets {
		isExist := false
		for _, sid := range m.senderIds {
			if tgt == sid {
				isExist = true
			}
		}
		if isExist == false {
			return false
		}
	}
	return true
}

func (m *Message) Set(a []*api.Agent, senderId uint64) {
	m.agents = append(m.agents, a...)
	m.senderIds = append(m.senderIds, senderId)
	if m.IsFinish() {
		logger.Info("closeChannel set agent")
		close(m.ready)
	}
}

func (m *Message) Get() []*api.Agent {
	select {
	case <-m.ready:
		//case <-time.After(100 * time.Millisecond):
		//	logger.Warn("Timeout Get")
	}
	logger.Info("Get agent")
	return m.agents
}

////////////////////////////////////////////////////////////
////////////           Harmovis server           ///////////
///////////////////////////////////////////////////////////

func runServer() *gosocketio.Server {

	currentRoot, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	d := filepath.Join(currentRoot, "monitor", "build")

	assetsDir = http.Dir(d)
	log.Println("AssetDir:", assetsDir)

	assetsDir = http.Dir(d)
	server := gosocketio.NewServer()

	server.On(gosocketio.OnConnection, func(c *gosocketio.Channel) {
		log.Printf("Connected from %s as %s", c.IP(), c.Id())
	})

	server.On(gosocketio.OnDisconnection, func(c *gosocketio.Channel) {
		log.Printf("Disconnected from %s as %s", c.IP(), c.Id())
	})

	return server
}

// assetsFileHandler for static Data
func assetsFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return
	}

	file := r.URL.Path
	//	log.Printf("Open File '%s'",file)
	if file == "/" {
		file = "/index.html"
	}
	f, err := assetsDir.Open(file)
	if err != nil {
		log.Printf("can't open file %s: %v\n", file, err)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		log.Printf("can't open file %s: %v\n", file, err)
		return
	}
	http.ServeContent(w, r, file, fi.ModTime(), f)
}

func runVisMonitor() {
	// Run HarmowareVis Monitor
	ioserv = runServer()
	log.Printf("Running Sio Server..\n")
	if ioserv == nil {
		os.Exit(1)
	}
	serveMux := http.NewServeMux()
	serveMux.Handle("/socket.io/", ioserv)
	serveMux.HandleFunc("/", assetsFileHandler)
	log.Printf("Starting Harmoware VIS  Provider on %s", *monitoraddr)
	err := http.ListenAndServe(*monitoraddr, serveMux)
	if err != nil {
		log.Fatal(err)
	}
}

/////////////////////////////////////////
////////  Send Agent to HamowareVIS /////
////////////////////////////////////////

type MapMarker struct {
	mtype int32   `json:"mtype"`
	id    int32   `json:"id"`
	lat   float32 `json:"lat"`
	lon   float32 `json:"lon"`
	angle float32 `json:"angle"`
	speed int32   `json:"speed"`
	area  int32   `json:"area"`
}

// GetJson: json化する関数
func (m *MapMarker) GetJson() string {
	s := fmt.Sprintf("{\"mtype\":%d,\"id\":%d,\"lat\":%f,\"lon\":%f,\"angle\":%f,\"speed\":%d,\"area\":%d}",
		m.mtype, m.id, m.lat, m.lon, m.angle, m.speed, m.area)
	return s
}

// sendAgentToHarmowareVis: harmowareVisに情報を送信する関数
func sendAgentToHarmowareVis(agents []*api.Agent) {

	if agents != nil {
		jsonAgents := make([]string, 0)
		for _, agentInfo := range agents {

			// agentInfoTypeによってエージェントを取得
			switch agentInfo.Type {
			case api.AgentType_PEDESTRIAN:
				//ped := agentInfo.GetPedestrian()
				mm := &MapMarker{
					mtype: int32(agentInfo.Type),
					id:    int32(agentInfo.Id),
					lat:   float32(agentInfo.Route.Position.Latitude),
					lon:   float32(agentInfo.Route.Position.Longitude),
					angle: float32(agentInfo.Route.Direction),
					speed: int32(agentInfo.Route.Speed),
				}
				jsonAgents = append(jsonAgents, mm.GetJson())

			case api.AgentType_CAR:
				//car := agentInfo.GetCar()
				mm := &MapMarker{
					mtype: int32(agentInfo.Type),
					id:    int32(agentInfo.Id),
					lat:   float32(agentInfo.Route.Position.Latitude),
					lon:   float32(agentInfo.Route.Position.Longitude),
					angle: float32(agentInfo.Route.Direction),
					speed: int32(agentInfo.Route.Speed),
				}
				jsonAgents = append(jsonAgents, mm.GetJson())
			}
		}
		mu.Lock()
		ioserv.BroadcastToAll("agents", jsonAgents)
		mu.Unlock()
	}
}

/////////////////////////////////////////
////////  Send Agent to HamowareVIS /////
////////////////////////////////////////

// sendAreaToHarmowareVis: harmowareVisに情報を送信する関数
func sendAreaToHarmowareVis(areas []*api.Area) {

	if areas != nil {
		jsonAreas := make([]string, 0)
		for _, areaInfo := range areas {
			areaJson, _ := json.Marshal(areaInfo)
			jsonAreas = append(jsonAreas, string(areaJson))
		}
		mu.Lock()
		ioserv.BroadcastToAll("areas", jsonAreas)
		mu.Unlock()
	}

}

// callbackForwardClockRequest: クロックを進める関数
func forwardClock() {
	/*// Databaseへ取得する
	targets := pm.GetProviderIds([]simutil.IDType{
		simutil.IDType_DATABASE,
	})
	//uid, _ := uuid.NewRandom()
	senderId := myProvider.Id
	sps, _ := visapi.GetAgentRequest(senderId, targets)
	//sps, _ := waiter.WaitSp(msgId, targets, 1000)

	allAgents := []*api.Agent{}
	for _, sp := range sps {
		agents := sp.GetSimSupply().GetGetAgentResponse().GetAgents()
		allAgents = append(allAgents, agents...)
	}*/

	agents := agentsMessage.Get()
	//agents := []*api.Agent{}

	// Harmowareに送る
	sendAgentToHarmowareVis(agents)

	logger.Info("Agents %v\n", len(agents))
}

////////////////////////////////////////////////////////////
////////////           Vis   Callback       ////////////////
///////////////////////////////////////////////////////////
type VisCallback struct {
	*util.Callback
}

func (cb *VisCallback) RegisterProviderRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) *api.Provider {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	p := simMsg.GetRegisterProviderRequest().GetProvider()
	pm.AddProvider(p)
	fmt.Printf("regist provider! %v %v\n", p.GetId(), p.GetType())

	// update provider to Vis
	targets := pm.GetProviderIds([]api.Provider_Type{})
	//sclient := sclientOpts[uint32(api.ChannelType_PROVIDER)].Sclient
	logger.Info("Send UpdateProvidersRequest %v, %v", targets, simapi.Provider)
	//simapi.UpdateProvidersRequest(sclient, targets, pm.GetProviders())
	logger.Info("Success Update Providers! Worker Num: ", len(targets))

	return simapi.Provider
}

func (cb *VisCallback) SetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	agents := simMsg.GetSetAgentRequest().GetAgents()
	logger.Info("get Agents: %v %v\n", len(agents), simMsg.GetSenderId())
	//sendAgentToHarmowareVis(agents)
	go agentsMessage.Set(agents, simMsg.GetSenderId())
	//db.Push(agents)
}

////////////////////////////////////////////////////////////
////////////     Vis Demand Supply Callback     ////////
///////////////////////////////////////////////////////////

/*func MbcbClockVis(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got clock callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
}

func MbcbProviderVis(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got provider callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_REGISTER_PROVIDER_REQUEST:
		// providerを追加する
		p := simMsg.GetRegisterProviderRequest().GetProvider()
		pm.AddProvider(p)
		fmt.Printf("regist request from agent of vis provider! %v\n", p)
		// 登録完了通知
		targets := []uint64{p.GetId()}
		sclient := sclientOptsVis[uint32(api.ChannelType_PROVIDER)].Sclient
		msgId := simMsg.GetMsgId()
		simapi.RegisterProviderResponse(sclient, msgId, myProvider)

		logger.Info("Success Regist Agent or Vis Providers", targets)
	}
}

func MbcbAgentVis(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got agent callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_SET_AGENT_REQUEST:

		agents := simMsg.GetSetAgentRequest().GetAgents()
		logger.Info("get Agents: %v %v\n", len(agents), simMsg.GetSenderId())
		//sendAgentToHarmowareVis(agents)
		go agentsMessage.Set(agents, simMsg.GetSenderId())
		//db.Push(agents)
		// response
		targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		sclient := sclientOptsVis[uint32(api.ChannelType_AGENT)].Sclient
		logger.Info("set Agent Finished: %v\n", len(agents))
		simapi.SetAgentResponse(sclient, msgId)
		logger.Info("set Agent Finished2: %v\n", len(agents))
	}
}

func MbcbAreaVis(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got mbcb callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
}*/

////////////////////////////////////////////////////////////
////////////           Master   Callback       ////////////////
///////////////////////////////////////////////////////////
type MasterCallback struct {
	*util.Callback
}

func (cb *MasterCallback) ForwardClockRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	fmt.Printf("get forwardClockRequest")
	//fmt.Printf("get forwardClockRequest")
	t1 := time.Now()
	forwardClock()
	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	logger.Info("Duration: %v, PID: %v", duration, myProvider.Id)
	// response to master
	//targets := []uint64{simMsg.GetSenderId()}
	msgId := simMsg.GetMsgId()
	sclient := sclientOptsMaster[uint32(api.ChannelType_CLOCK)].Sclient
	logger.Debug("Response to master pid %v, msgId%v\n", myProvider.Id, msgId)
	simapi.ForwardClockResponse(sclient, msgId)

	// 初期化
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	agentsMessage = NewMessage(targets)
}

func (cb *MasterCallback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	providers := simMsg.GetUpdateProvidersRequest().GetProviders()
	logger.Info("Finish: Update Workers num: %v\n", len(providers))
}

func (cb *MasterCallback) SendAreaInfoRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	areas := simMsg.GetSendAreaInfoRequest().GetAreas()
	sendAreaToHarmowareVis(areas)
}

func (cb *MasterCallback) SetClockRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)

	logger.Info("Finish: Set Clock %v\n")
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
		//fmt.Printf("get forwardClockRequest")
		t1 := time.Now()
		forwardClock()
		t2 := time.Now()
		duration := t2.Sub(t1).Milliseconds()
		logger.Info("Duration: %v, PID: %v", duration, myProvider.Id)
		// response to master
		targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		sclient := sclientOptsMaster[uint32(api.ChannelType_CLOCK)].Sclient
		logger.Debug("Response to master pid %v, msgId%v\n", myProvider.Id, msgId)
		simapi.ForwardClockResponse(sclient, msgId)

		// 初期化
		targets = pm.GetProviderIds([]api.Provider_Type{
			api.Provider_AGENT,
		})
		agentsMessage = NewMessage(targets)
	}
}

func MbcbProviderMaster(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got provider callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_REGISTER_PROVIDER_RESPONSE:
		simapi.SendMsgToWait(msg)
		fmt.Printf("regist provider to Master Provider!\n")
	case api.MsgType_UPDATE_PROVIDERS_REQUEST:
		providers := simMsg.GetUpdateProvidersRequest().GetProviders()
		//pm.SetProviders(providers) worker情報はいらない
		targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		sclient := sclientOptsMaster[uint32(api.ChannelType_PROVIDER)].Sclient
		simapi.UpdateProvidersResponse(sclient, msgId)
		logger.Info("Finish: Update Workers num: %v\n", len(providers))
	}
}

func MbcbAgentMaster(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got agent callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
}

func MbcbAreaMaster(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got mbcb callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_SEND_AREA_INFO_REQUEST:
		areas := simMsg.GetSendAreaInfoRequest().GetAreas()
		sendAreaToHarmowareVis(areas)
		// response
		targets := []uint64{simMsg.GetSenderId()}
		msgId := simMsg.GetMsgId()
		sclient := sclientOptsMaster[uint32(api.ChannelType_AREA)].Sclient
		simapi.SendAreaInfoResponse(sclient, msgId)
		logger.Info("Finish: GetAreaInfo: %v\n", areas)
	}
}*/

func main() {
	logger.Info("StartUp Provider %v, %v", *servaddr, myProvider)
	fmt.Printf("NumCPU=%d\n", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Printf("Start Vis Provider")
	// Connect to Worker Syenrex Node Server
	// Register Node Server

	channelTypes := []uint32{}
	for _, opt := range sclientOptsVis {
		channelTypes = append(channelTypes, opt.ChType)
	}
	ni := sxutil.GetDefaultNodeServInfo()
	util.RegisterNodeLoop(ni, *nodeaddr, "VisProvider", channelTypes)

	// Register Synerex Server
	client := util.RegisterSynerexLoop(*servaddr)
	util.RegisterSXServiceClients(ni, client, sclientOptsVis)
	logger.Info("Register Synerex Server")

	// Connect to Master Syenrex Node Server
	// Register Node Server
	channelTypes = []uint32{}
	for _, opt := range sclientOptsMaster {
		channelTypes = append(channelTypes, opt.ChType)
	}
	ni = sxutil.NewNodeServInfo()
	util.RegisterNodeLoop(ni, *masterNodeaddr, "VisProvider", channelTypes)

	// Register Synerex Server
	client = util.RegisterSynerexLoop(*masterServaddr)
	util.RegisterSXServiceClients(ni, client, sclientOptsMaster)
	logger.Info("Register Synerex Server")

	wg := sync.WaitGroup{} // for syncing other goroutines
	wg.Add(1)

	sclient := sclientOptsMaster[uint32(api.ChannelType_PROVIDER)].Sclient
	masterProvider = util.RegisterProviderLoop(sclient, simapi)
	logger.Info("Register Master Provider")

	runVisMonitor()

	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!

}
