package main

import (
	//"context"

	"flag"
	"strconv"

	//"fmt"
	//"log"

	//"math/rand"
	"os"
	"sync"

	"math/rand"
	"time"

	//"runtime"
	"encoding/json"
	"runtime"

	api "github.com/RuiHirano/flow_beta/api"
	algo "github.com/RuiHirano/flow_beta/provider/agent/algorithm"
	util "github.com/RuiHirano/flow_beta/util"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	sxapi "github.com/synerex/synerex_api"
	sxutil "github.com/synerex/synerex_sxutil"
)

var (
	myProvider     *api.Provider
	workerProvider *api.Provider
	visProvider    *api.Provider
	pm             *api.ProviderManager
	sim            *Simulator
	logger         *util.Logger
	mu             sync.Mutex
	myArea         *api.Area
	agentType      api.AgentType
	agentProvider  *AgentProvider
	recorder *Recorder

	simapi            *api.SimAPI

	servaddr     = flag.String("servaddr", getServerAddress(), "The Synerex Server Listening Address")
	nodeaddr     = flag.String("nodeaddr", getNodeservAddress(), "Node ID Server Address")
	visServaddr  = flag.String("visServaddr", getVisServerAddress(), "Vis Synerex Server Listening Address")
	visNodeaddr  = flag.String("visNodeaddr", getVisNodeservAddress(), "Vis Node ID Server Address")
	providerName = flag.String("providerName", getProviderName(), "Provider Name")
	areaJson     = flag.String("areaJson", getAreaJson(), "Area Information")
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

func getVisNodeservAddress() string {
	env := os.Getenv("SX_VIS_NODESERV_ADDRESS")
	if env != "" {
		return env
	} else {
		return "127.0.0.1:9990"
	}
}

func getVisServerAddress() string {
	env := os.Getenv("SX_VIS_SERVER_ADDRESS")
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
		return "AgentProvider"
	}
}

func getAreaJson() string {
	env := os.Getenv("AREA_JSON")
	if env != "" {
		return env
	} else {
		return ""
	}
}

func GetMockAgents(agentNum uint64) []*api.Agent {
	agents := make([]*api.Agent, 0)
	minLon, maxLon, minLat, maxLat := 136.971626, 136.989379, 35.152210, 35.161499
	//maxLat, maxLon, minLat, minLon := GetCoordRange(proc.Area.ControlArea)
	//fmt.Printf("minLon %v, maxLon %v, minLat %v, maxLat %v\n", minLon, maxLon, minLat, maxLat)
	for i := 0; i < int(agentNum); i++ {
		uid, _ := uuid.NewRandom()
		position := &api.Coord{
			Longitude: minLon + (maxLon-minLon)*rand.Float64(),
			Latitude:  minLat + (maxLat-minLat)*rand.Float64(),
		}
		transitPoint := &api.TransitPoint{
			Id: string(uid.ID()),
			Coord:&api.Coord{
			Longitude: minLon + (maxLon-minLon)*rand.Float64(),
			Latitude:  minLat + (maxLat-minLat)*rand.Float64(),
		},
	}

		transitPoints := []*api.TransitPoint{transitPoint}
		var agentParam  *algo.AgentParam
		if rand.Intn(2) != 0{
			//logger.Info("true", rand.Intn(2) != 0)
			agentParam = &algo.AgentParam{
				Status: "I",
				Move: 0,
			}
		}else{
			//logger.Info("false", rand.Intn(2) != 0)
			agentParam = &algo.AgentParam{
				Status: "S",
				Move: 0,
			}
		}
		agentModelParamJson, _ := json.Marshal(agentParam)
		agents = append(agents, &api.Agent{
			Type: api.AgentType_PEDESTRIAN,
			Id:   uint64(uid.ID()),
			Route: &api.Route{
				Position:      position,
				Direction:     30,
				Speed:         60,
				TransitPoints: transitPoints,
				NextTransit:   transitPoint,
			},
			Data: string(agentModelParamJson),
		})
		//fmt.Printf("position %v\n", position)
	}

	return agents
}

func init() {
	flag.Parse()
	logger = util.NewLogger()
	//agents := GetMockAgents(10)
	//s, _ := json.Marshal(agents)
	//logger.Debug("jsonagents %v", string(s))

	recorder = NewRecorder()
	/*recorder.Add(GetMockAgents(10))
	recorder.Add(GetMockAgents(10))
	recorder.Add(GetMockAgents(10))
	recorder.Add(GetMockAgents(10))
	recorder.Add(GetMockAgents(10))
	recorder.Add(GetMockAgents(10))
	recorder.Add(GetMockAgents(10))
	recorder.Add(GetMockAgents(10))
	recorder.Add(GetMockAgents(10))*/

	//areaJson := os.Getenv("AREA")
	bytes := []byte(*areaJson)
	json.Unmarshal(bytes, &myArea)
	//fmt.Printf("myArea: %v\n", myArea)

	agentType = api.AgentType_PEDESTRIAN
}

type StepData struct {
	Time int  `json:"time"`
	Infected int   `json:"infected"`
	Susceptible int   `json:"susceptible"`
	Recovered int    `json:"recovered"`
}
type Recorder struct{
	Datetime string
	Data[]*StepData
	SaveNum int
	Time int
}
func NewRecorder() *Recorder{
	return &Recorder{
		Datetime: time.Now().Format("20060102-1504"),
		Data: []*StepData{},
		SaveNum: 0,
		Time: 0,
	}
}

func (rc *Recorder) Add(agents []*api.Agent) error {
	stepData := &StepData{}
	stepData.Time = rc.Time
	rc.Time += 1
	for _, agent := range agents{
		var data *algo.AgentParam
		json.Unmarshal([]byte(agent.Data), &data)
		if data.Status == "S"{
			stepData.Susceptible += 1
		}else if data.Status == "I"{
			stepData.Infected += 1
		}else if data.Status == "R"{
			stepData.Recovered += 1
		}
	}
	rc.Data = append(rc.Data, stepData)
	logger.Info("", len(rc.Data))
	if len(rc.Data) > 100{
		rc.Save()
		rc.Data = []*StepData{}
		rc.SaveNum +=1
	}
	return nil
}

func (rc *Recorder) CheckDir (dirname string) {
	if _, err := os.Stat("./result/"+dirname); os.IsNotExist(err) {
	  os.Mkdir("./result/"+dirname, 0777)
	}
  }
  

func (rc *Recorder) Save() error {
	rc.CheckDir(rc.Datetime)
	name := *providerName+"_"+strconv.Itoa(rc.SaveNum)+".json"
	file, err := os.Create("./result/"+rc.Datetime + "/" +name)
    if err != nil {
        logger.Error("ERROR: %v", err)
    }
    defer file.Close()

	jsonData, _ := json.Marshal(rc.Data)
    file.Write(([]byte)(string(jsonData)))            // ファイル出力
	return nil
}

////////////////////////////////////////////////////////////
////////////     Agent Provider        ////////////////
///////////////////////////////////////////////////////////
type AgentProvider struct {
	Simulator *Simulator
	WorkerAPI *api.ProviderAPI
	VisAPI *api.ProviderAPI
}

func NewAgentProvider(simulator *Simulator, workerapi *api.ProviderAPI, visapi *api.ProviderAPI) *AgentProvider {
	ap := &AgentProvider{
		Simulator: simulator,
		WorkerAPI: workerapi,
		VisAPI: visapi,
	}
	return ap
}

func (ap *AgentProvider) Connect() error {
	ap.WorkerAPI.ConnectServer(false)
	ap.WorkerAPI.RegisterProvider()
	// For without Vis
	//ap.VisAPI.ConnectServer()
	//ap.VisAPI.RegisterProvider()
	return nil
}

// 
func (ap *AgentProvider) SimulatorRequest(simReq *api.SimulatorRequest) {
	ap.Simulator.SimulationRequest(simReq)
	logger.Success("SimulationRequest")
}

// 
func (ap *AgentProvider) Reset() {
	ap.Simulator.ResetAgents()
	logger.Success("Reset")
}

// Connect: Worker Nodeに接続する
func (ap *AgentProvider) ForwardClock() error {
	//logger.Debug("calcNextAgents 0")
	t1 := time.Now()
	ap.Simulator.ForwardStep() // agents in control area

	//logger.Debug("calcNextAgents 2")
	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("CalcNextAgents! Duration: %v ms, Wait: %d ms", duration, interval-duration)
	}
	return nil
}

func (ap *AgentProvider) GetSameAreaAgents() []*api.Agent {
	//logger.Debug("getSameAreaAgents 0")
	t1 := time.Now()
	//logger.Debug("1: 同エリアエージェント取得")
	targets := pm.GetTargets([]api.Provider_Type{
		api.Provider_AGENT,
	})
	sameAgents, _ := ap.WorkerAPI.GetAgents(targets)
	sim.SetDiffAgents(sameAgents)
	logger.Success("GetSameAgents: %d %v", len(sameAgents), targets)
	
	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("GetSameAreaAgents! Duration: %v ms, Wait: %d ms", duration, interval-duration)
	}
	return sameAgents
}


func (ap *AgentProvider) GetNeighborAreaAgents() []*api.Agent {
	//logger.Debug("getSameAreaAgents 0")
	t1 := time.Now()
	//logger.Debug("1: 同エリアエージェント取得")
	targets := pm.GetTargets([]api.Provider_Type{
		api.Provider_GATEWAY,
	})
	neighborAgents, _ := ap.WorkerAPI.GetAgents(targets)
	sim.SetDiffAgents(neighborAgents)
	logger.Success("GetNeighborAgents: %d", len(neighborAgents))
	
	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("GetNeighborAgents! Duration: %v ms, Wait: %d ms", duration, interval-duration)
	}
	return neighborAgents
}



func (ap *AgentProvider) UpdateAgents(agents []*api.Agent) error {
	t1 := time.Now()
	
	newAgents := sim.UpdateDuplicateAgents(agents)
	// Agentsをセットする
	sim.SetAgents(newAgents)

	// resultにfileとして保存する
	//recorder.Add(newAgents)

	//logger.Debug("updateNextAgents 3")
	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Warn("time cycle delayed... Duration: %d", duration)
	} else {
		logger.Success("UpdateNextAgents! Duration: %v ms, Wait: %d ms", duration, interval-duration)
	}
	return nil
}

func (ap *AgentProvider) GetAgents() []*api.Agent {

	return ap.Simulator.Agents
}

func (ap *AgentProvider) AddAgents(agents []*api.Agent) int {
	//logger.Info("agents num %v", len(agents))
	ap.Simulator.AddAgents(agents)
	// FIX
	return len(agents)
}

////////////////////////////////////////////////////////////
////////////         Worker Callback       ////////////////
///////////////////////////////////////////////////////////
type WorkerCallback struct {
	*api.Callback
}

func (cb *WorkerCallback) SimulatorRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	simReq := simMsg.GetSimulatorRequest()
	agentProvider.SimulatorRequest(simReq)
	logger.Success("Simulator\n")
}

func (cb *WorkerCallback) ResetRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	agentProvider.Reset()
	logger.Success("Reset\n")
}

func (cb *WorkerCallback) ForwardClockInitRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	agentProvider.GetSameAreaAgents()
	logger.Success("Forward Clock Init")
}

func (cb *WorkerCallback) ForwardClockMainRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	agentProvider.ForwardClock()
	logger.Success("Forward Clock Main Agents: %d", len(sim.Agents))
}

func (cb *WorkerCallback) ForwardClockTerminateRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	//agents := agentProvider.GetNeighborAreaAgents()
	//agentProvider.UpdateAgents(agents)
	logger.Success("Forward Clock Terminate")
}

func (cb *WorkerCallback) UpdateProvidersRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	providers := simMsg.GetUpdateProvidersRequest().GetProviders()
	pm.SetProviders(providers)
	logger.Success("Update Workers num: %d\n", len(providers))
}

func (cb *WorkerCallback) SetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)

	// Agentをセットする
	agents := simMsg.GetSetAgentRequest().GetAgents()
	//logger.Success("Set Agent NUM %d", agents)
	// for experiment
	//expAgents := CreateExperimentAgents(agents)

	// Agent情報を追加する
	num := agentProvider.AddAgents(agents)
	logger.Success("Set Agent %d", num)
}

func (cb *WorkerCallback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*api.Agent {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	//logger.Debug("GetAgents Worker: %v")
	agents := agentProvider.GetAgents()
	logger.Success("Send %d Agent to %d", len(agents), simMsg.SenderId)
	return agents
}

////////////////////////////////////////////////////////////
////////////          Vis Callback         ////////////////
///////////////////////////////////////////////////////////
type VisCallback struct {
	*api.Callback
}

func (cb *VisCallback) GetAgentRequest(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) []*api.Agent {
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	logger.Debug("GetAgents VIS")
	agents := agentProvider.GetAgents()
	logger.Success("Send %d Agent to VIS %d", len(agents), simMsg.SenderId)
	return agents
}

func main() {
	logger.Info("Start Agent Provider")
	logger.Info("NumCPU=%d", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())

	wg := sync.WaitGroup{} // for syncing other goroutines
	wg.Add(1)

	// Simulator
	sim = NewSimulator(myArea, api.AgentType_PEDESTRIAN)

	// Worker Server
	uid, _ := uuid.NewRandom()
	myProvider := &api.Provider{
		Id:   uint64(uid.ID()),
		Name: "AgentProvider",
		Type: api.Provider_AGENT,
	}
	pm = api.NewProviderManager(myProvider)
	cb := api.NewCallback()

	// Worker Server
	wocb := &WorkerCallback{cb} // override
	workerAPI := api.NewProviderAPI(myProvider, *servaddr, *nodeaddr, wocb)
	//workerAPI.ConnectServer()
	//workerAPI.RegisterProvider()

	// Vis Server
	vicb := &VisCallback{cb} // override
	visAPI := api.NewProviderAPI(myProvider, *visServaddr, *visNodeaddr, vicb)
	//visAPI.ConnectServer()
	//visAPI.RegisterProvider()

	// AgentProvider
	agentProvider = NewAgentProvider(sim, workerAPI, visAPI)
	agentProvider.Connect()

	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!
	logger.Success("Terminate Agent Provider")
}
