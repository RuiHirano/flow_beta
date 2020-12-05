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
	visProvider *VisualizationProvider

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

	logger = util.NewLogger()
}

////////////////////////////////////////////////////////////
////////////     Visualization Provider     ////////////////
///////////////////////////////////////////////////////////
type VisualizationProvider struct {
	MasterAPI *util.MasterAPI
	WorkerAPI *util.WorkerAPI
}

func NewVisualizationProvider(masterapi *util.MasterAPI, workerapi *util.WorkerAPI) *VisualizationProvider {
	ap := &VisualizationProvider{
		MasterAPI: masterapi,
		WorkerAPI: workerapi,
	}
	return ap
}

func (ap *VisualizationProvider) Connect() error {
	ap.WorkerAPI.ConnectServer()
	//ap.WorkerAPI.RegisterProvider()
	ap.MasterAPI.ConnectServer()
	ap.MasterAPI.RegisterProvider()
	return nil
}

// 
func (ap *VisualizationProvider) RegisterProvider(provider *api.Provider) error {
	//logger.Debug("calcNextAgents 0")
	pm.AddProvider(provider)
	//fmt.Printf("regist provider! %v %v\n", p.GetId(), p.GetType())

	logger.Debug("RegisterProvider: %v", provider)
	// update provider to worker
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	providers := pm.GetProviders()
	ap.WorkerAPI.UpdateProviders(targets, providers)
	logger.Success("Update Providers! Worker Num: ", len(targets))
	return nil
}

// 
func (ap *VisualizationProvider) GetAgents() []*api.Agent {
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_AGENT,
	})
	logger.Debug("targets: %v", targets)
	agents := ap.WorkerAPI.GetAgents(targets)
	logger.Success("Get Agents : %s", len(agents))
	return agents
}

// 
func (ap *VisualizationProvider) SendAgentsToMonitor(agents []*api.Agent) error {
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
	return nil
}

// 
func (ap *VisualizationProvider) SendAreaToMonitor(areas []*api.Area) error {
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
	return nil
}

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

////////////////////////////////////////////////////////////
////////////           Worker   Callback       ////////////////
///////////////////////////////////////////////////////////
type WorkerCallback struct {
	*util.Callback
}

func (cb *WorkerCallback) RegisterProviderRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) *api.Provider {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	provider := simMsg.GetRegisterProviderRequest().GetProvider()
	visProvider.RegisterProvider(provider)
	return visProvider.WorkerAPI.SimAPI.Provider
}


////////////////////////////////////////////////////////////
////////////           Master   Callback       ////////////////
///////////////////////////////////////////////////////////
type MasterCallback struct {
	*util.Callback
}

func (cb *MasterCallback) ForwardClockTerminateRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	t1 := time.Now()
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	agents := visProvider.GetAgents()
	visProvider.SendAgentsToMonitor(agents)

	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("Forward Clock Terminate! Duration: %v ms, Wait: %d ms", duration, interval-duration)
	}
}

func (cb *MasterCallback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	providers := simMsg.GetUpdateProvidersRequest().GetProviders()
	logger.Success("Update Workers num: %d\n", len(providers))
}

func (cb *MasterCallback) SendAreaInfoRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	areas := simMsg.GetSendAreaInfoRequest().GetAreas()
	visProvider.SendAreaToMonitor(areas)
	logger.Success("Send Area Info")
}

func main() {
	logger.Info("Start Visualization Provider")
	logger.Info("NumCPU=%d", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())

	wg := sync.WaitGroup{} // for syncing other goroutines
	wg.Add(1)

	go runVisMonitor()

	// Vis
	uid, _ := uuid.NewRandom()
	myProvider := &api.Provider{
		Id:   uint64(uid.ID()),
		Name: "VisProvider",
		Type: api.Provider_VISUALIZATION,
	}
	pm = util.NewProviderManager(myProvider)
	simapi := api.NewSimAPI(myProvider)
	cb := util.NewCallback()

	// Worker Server
	wocb := &WorkerCallback{cb} // override
	workerAPI := util.NewWorkerAPI(simapi, *servaddr, *nodeaddr, wocb)
	//workerAPI.ConnectServer()
	//workerAPI.RegisterProvider()

	// Master Server
	macb := &MasterCallback{cb} // override
	masterAPI := util.NewMasterAPI(simapi, *masterServaddr, *masterNodeaddr, macb)
	//masterAPI.ConnectServer()
	//masterAPI.RegisterProvider()

	// VisualizationProvider
	visProvider = NewVisualizationProvider(masterAPI, workerAPI)
	visProvider.Connect()


	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!
	logger.Success("Terminate Visualization Provider")
}
