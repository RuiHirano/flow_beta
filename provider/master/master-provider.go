package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"io/ioutil"
	"os/exec"

	api "github.com/RuiHirano/flow_beta/api"
	util "github.com/RuiHirano/flow_beta/util"
	"github.com/go-yaml/yaml"
	"github.com/golang/protobuf/proto"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	sxapi "github.com/synerex/synerex_api"
	sxutil "github.com/synerex/synerex_sxutil"
)

var (
	myProvider  *api.Provider
	synerexAddr string
	nodeIdAddr  string
	port        string
	startFlag   bool
	masterClock int
	mu          sync.Mutex
	//providerManager *Manager
	pm     *util.ProviderManager
	logger *util.Logger
	config *Config
	podgen *PodGenerator
	proc   *Processor

	sclientOpts map[uint32]*util.SclientOpt
	simapi      *api.SimAPI
)

type Config struct {
	Area Config_Area `yaml:"area"`
}

type Config_Area struct {
	SideRange      float64        `yaml:"sideRange"`
	DuplicateRange float64        `yaml:"duplicateRange"`
	DefaultAreaNum Config_AreaNum `yaml:"defaultAreaNum"`
}
type Config_AreaNum struct {
	Row    uint64 `yaml:"row"`
	Column uint64 `yaml:"column"`
}

func readConfig() (*Config, error) {
	var config *Config
	buf, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		fmt.Println(err)
		return config, err
	}
	// []map[string]string のときと使う関数は同じです。
	// いい感じにマッピングしてくれます。
	err = yaml.Unmarshal(buf, &config)
	if err != nil {
		fmt.Println(err)
		return config, err
	}
	fmt.Printf("yaml is %v\n", config)
	return config, nil
}

func init() {

	sclientOpts = map[uint32]*util.SclientOpt{
		uint32(api.ChannelType_CLOCK): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_CLOCK),
			MBusCallback: MbcbClock,
			ArgJson:      fmt.Sprintf("{Client:MasterProvider_Clock}"),
		},
		uint32(api.ChannelType_PROVIDER): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_PROVIDER),
			MBusCallback: MbcbProvider,
			ArgJson:      fmt.Sprintf("{Client:MasterProvider_Provider}"),
		},
		uint32(api.ChannelType_AGENT): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AGENT),
			MBusCallback: MbcbAgent,
			ArgJson:      fmt.Sprintf("{Client:MasterProvider_Agent}"),
		},
		uint32(api.ChannelType_AREA): &util.SclientOpt{
			ChType:       uint32(api.ChannelType_AREA),
			MBusCallback: MbcbArea,
			ArgJson:      fmt.Sprintf("{Client:MasterProvider_Area}"),
		},
	}
	uid, _ := uuid.NewRandom()
	myProvider := &api.Provider{
		Id:   uint64(uid.ID()),
		Name: "MasterServer",
		Type: api.Provider_MASTER,
	}
	simapi = api.NewSimAPI(myProvider)
	pm = util.NewProviderManager(myProvider)
	log.Printf("ProviderID: %d", simapi.Provider.Id)

	podgen = NewPodGenerator()
	proc = NewProcessor()
	startFlag = false
	masterClock = 0
	logger = util.NewLogger()
	logger.SetPrefix("Master")
	flag.Parse()
	//providerManager = NewManager()
	// configを読み取る
	config, _ = readConfig()

	synerexAddr = os.Getenv("SYNEREX_SERVER")
	if synerexAddr == "" {
		synerexAddr = "127.0.0.1:10000"
	}
	nodeIdAddr = os.Getenv("NODEID_SERVER")
	if nodeIdAddr == "" {
		nodeIdAddr = "127.0.0.1:9000"
	}
	port = os.Getenv("PORT")
	if port == "" {
		port = "9990"
	}
}

////////////////////////////////////////////////////////////
//////////////////        Manager          ///////////////////
///////////////////////////////////////////////////////////

/*type Manager struct {
	Providers []*api.Provider
}

func NewManager() *Manager {
	m := &Manager{
		Providers: make([]*api.Provider, 0),
	}
	return m
}

func (m *Manager) AddProvider(provider *api.Provider) {
	m.Providers = append(m.Providers, provider)
}

func (m *Manager) GetProviderIds() []uint64 {
	pids := make([]uint64, 0)
	for _, p := range m.Providers {
		pids = append(pids, p.GetId())
	}
	return pids
}*/

////////////////////////////////////////////////////////////
////////////     Demand Supply Callback     ////////////////
///////////////////////////////////////////////////////////

func MbcbClock(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got clock callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {

	case api.MsgType_SET_CLOCK_RESPONSE:
		simapi.SendMsgToWait(msg)
	case api.MsgType_FORWARD_CLOCK_RESPONSE:
		simapi.SendMsgToWait(msg)
	}
}

func MbcbProvider(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got provider callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_REGISTER_PROVIDER_REQUEST:
		// Register Provider Request from Worker
		// providerを追加する
		// TODO
		p := simMsg.GetRegisterProviderRequest().GetProvider()
		pm.AddProvider(p)
		fmt.Printf("regist provider! %v\n", p.GetId())
		// 登録完了通知
		targets := []uint64{p.GetId()}
		msgId := simMsg.GetMsgId()
		sclient := sclientOpts[uint32(api.ChannelType_PROVIDER)].Sclient
		simapi.RegisterProviderResponse(sclient, targets, msgId, simapi.Provider)

		// update provider to worker
		targets = pm.GetProviderIds([]api.Provider_Type{
			api.Provider_WORKER,
		})
		simapi.UpdateProvidersRequest(sclient, targets, pm.GetProviders())

		logger.Info("Success Update Providers! Worker Num: ", len(targets))

	case api.MsgType_UPDATE_PROVIDERS_RESPONSE:
		simapi.SendMsgToWait(msg)
	}
}

func MbcbAgent(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got agent callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_SET_AGENT_RESPONSE:
		simapi.SendMsgToWait(msg)
	}
}

func MbcbArea(clt *sxutil.SXServiceClient, msg *sxapi.MbusMsg) {
	log.Println("Got mbcb callback")
	simMsg := &api.SimMsg{}
	proto.Unmarshal(msg.GetCdata().GetEntity(), simMsg)
	switch simMsg.GetType() {
	case api.MsgType_SEND_AREA_INFO_RESPONSE:
		simapi.SendMsgToWait(msg)
	}
}

/*// Supplyのコールバック関数
func supplyCallback(clt *api.SMServiceClient, sp *api.Supply) {
	// 自分宛かどうか
	// check if supply is match with my demand.
	switch sp.GetSimSupply().GetType() {
	case api.SupplyType_SET_CLOCK_RESPONSE:
		simapi.SendSpToWait(sp)
	case api.SupplyType_SET_AGENT_RESPONSE:
		simapi.SendSpToWait(sp)
	case api.SupplyType_FORWARD_CLOCK_RESPONSE:
		simapi.SendSpToWait(sp)
	case api.SupplyType_FORWARD_CLOCK_INIT_RESPONSE:
		simapi.SendSpToWait(sp)
	case api.SupplyType_UPDATE_PROVIDERS_RESPONSE:
		simapi.SendSpToWait(sp)
	case api.SupplyType_SEND_AREA_INFO_RESPONSE:
		simapi.SendSpToWait(sp)
	}
}*/

// Demandのコールバック関数
/*func demandCallback(clt *api.SMServiceClient, dm *api.Demand) {

	// check if supply is match with my demand.
	switch dm.GetSimDemand().GetType() {
	case api.DemandType_REGIST_PROVIDER_REQUEST:
		// providerを追加する
		p := dm.GetSimDemand().GetRegistProviderRequest().GetProvider()
		//providerManager.AddProvider(p)
		pm.AddProvider(p)
		fmt.Printf("regist provider! %v\n", p.GetId())
		// 登録完了通知
		//targets := []uint64{tid}
		senderInfo := myProvider.Id
		targets := []uint64{p.GetId()}
		msgId := dm.GetSimDemand().GetMsgId()
		simapi.RegistProviderResponse(senderInfo, targets, msgId, pm.MyProvider)

		// update provider to worker
		targets = pm.GetProviderIds([]api.Provider_Type{
			api.Provider_WORKER,
		})
		simapi.UpdateProvidersRequest(senderInfo, targets, pm.GetProviders())

		logger.Info("Success Update Providers! Worker Num: ", len(targets))

	}
}*/

///////////////////////////////////////////////
////////////  Processor  //////////////////////
///////////////////////////////////////////////

type Processor struct {
	Area        *api.Area            // 全体のエリア
	AreaMap     map[string]*api.Area // [areaid] []areaCoord     エリア情報を表したmap
	NeighborMap map[string][]string  // [areaid] []neighborAreaid   隣接関係を表したmap
}

func NewProcessor() *Processor {
	proc := &Processor{
		Area:        nil,
		AreaMap:     make(map[string]*api.Area),
		NeighborMap: make(map[string][]string),
	}
	return proc
}

// setAgents3: gatewayから入れる
// routes から東山のrouteを作成する
func (proc *Processor) setAgents3(agentNum uint64) (bool, error) {

	if proc.Area == nil {
		return false, fmt.Errorf("area is nil")
	}

	agents := make([]*api.Agent, 0)
	//minLon, maxLon, minLat, maxLat := 136.971626, 136.989379, 35.152210, 35.161499
	//maxLat, maxLon, minLat, minLon := GetCoordRange(proc.Area.ControlArea)
	//fmt.Printf("minLon %v, maxLon %v, minLat %v, maxLat %v\n", minLon, maxLon, minLat, maxLat)
	for i := 0; i < int(agentNum); i++ {
		uid, _ := uuid.NewRandom()
		routes := GetRoutes()
		route1 := routes[0]
		point1 := route1.Point
		point2 := route1.NeighborPoints[rand.Intn(int(len(route1.NeighborPoints)))].Point
		position := GetAmongPosition(point1, point2)
		nextTransit := point2

		agents = append(agents, &api.Agent{
			Type: api.AgentType_PEDESTRIAN,
			Id:   uint64(uid.ID()),
			Route: &api.Route{
				Position:      position,
				Direction:     30,
				Speed:         60,
				Departure:     position,
				Destination:   position,
				TransitPoints: []*api.Coord{},
				NextTransit:   nextTransit,
			},
		})
		fmt.Printf("position %v\n", position)
	}

	// エージェントを設置するリクエスト
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_WORKER,
	})
	sclient := sclientOpts[uint32(api.ChannelType_AGENT)].Sclient
	simapi.SetAgentRequest(sclient, targets, agents)

	logger.Info("Finish Setting Agents \n Add: %v", len(agents))
	return true, nil
}

// setAgents: agentをセットするDemandを出す関数
// routes から東山のrouteを作成する
func (proc *Processor) setAgents2(agentNum uint64) (bool, error) {

	if proc.Area == nil {
		return false, fmt.Errorf("area is nil")
	}

	agents := make([]*api.Agent, 0)
	//minLon, maxLon, minLat, maxLat := 136.971626, 136.989379, 35.152210, 35.161499
	maxLat, maxLon, minLat, minLon := GetCoordRange(proc.Area.ControlArea)
	fmt.Printf("minLon %v, maxLon %v, minLat %v, maxLat %v\n", minLon, maxLon, minLat, maxLat)
	for i := 0; i < int(agentNum); i++ {
		uid, _ := uuid.NewRandom()
		routes := GetRoutes()
		route1 := routes[rand.Intn(len(routes))]
		point1 := route1.Point
		point2 := route1.NeighborPoints[rand.Intn(int(len(route1.NeighborPoints)))].Point
		position := GetAmongPosition(point1, point2)
		nextTransit := point2

		agents = append(agents, &api.Agent{
			Type: api.AgentType_PEDESTRIAN,
			Id:   uint64(uid.ID()),
			Route: &api.Route{
				Position:      position,
				Direction:     30,
				Speed:         60,
				Departure:     position,
				Destination:   position,
				TransitPoints: []*api.Coord{},
				NextTransit:   nextTransit,
			},
		})
		fmt.Printf("position %v\n", position)
	}

	// エージェントを設置するリクエスト
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_WORKER,
	})
	sclient := sclientOpts[uint32(api.ChannelType_AGENT)].Sclient
	simapi.SetAgentRequest(sclient, targets, agents)

	logger.Info("Finish Setting Agents \n Add: %v", len(agents))
	return true, nil
}

// setAgents: agentをセットするDemandを出す関数
func (proc *Processor) setAgents(agentNum uint64) (bool, error) {

	if proc.Area == nil {
		return false, fmt.Errorf("area is nil")
	}

	agents := make([]*api.Agent, 0)
	//minLon, maxLon, minLat, maxLat := 136.971626, 136.989379, 35.152210, 35.161499
	maxLat, maxLon, minLat, minLon := GetCoordRange(proc.Area.ControlArea)
	fmt.Printf("minLon %v, maxLon %v, minLat %v, maxLat %v\n", minLon, maxLon, minLat, maxLat)
	for i := 0; i < int(agentNum); i++ {
		uid, _ := uuid.NewRandom()
		position := &api.Coord{
			Longitude: minLon + (maxLon-minLon)*rand.Float64(),
			Latitude:  minLat + (maxLat-minLat)*rand.Float64(),
		}
		destination := &api.Coord{
			Longitude: minLon + (maxLon-minLon)*rand.Float64(),
			Latitude:  minLat + (maxLat-minLat)*rand.Float64(),
		}
		transitPoint := &api.Coord{
			Longitude: minLon + (maxLon-minLon)*rand.Float64(),
			Latitude:  minLat + (maxLat-minLat)*rand.Float64(),
		}

		transitPoints := []*api.Coord{transitPoint}
		agents = append(agents, &api.Agent{
			Type: api.AgentType_PEDESTRIAN,
			Id:   uint64(uid.ID()),
			Route: &api.Route{
				Position:      position,
				Direction:     30,
				Speed:         60,
				Departure:     position,
				Destination:   destination,
				TransitPoints: transitPoints,
				NextTransit:   transitPoint,
			},
		})
		//fmt.Printf("position %v\n", position)
	}

	// エージェントを設置するリクエスト
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_WORKER,
	})
	sclient := sclientOpts[uint32(api.ChannelType_AGENT)].Sclient
	simapi.SetAgentRequest(sclient, targets, agents)

	logger.Info("Finish Setting Agents \n Add: %v", len(agents))
	return true, nil
}

// setAreas: areaをセットするDemandを出す関数
func (proc *Processor) setAreas(areaCoords []*api.Coord) (bool, error) {

	proc.Area = &api.Area{
		Id:            0,
		ControlArea:   areaCoords,
		DuplicateArea: areaCoords,
	}
	//id := "test"

	areas, neighborsMap := proc.divideArea(areaCoords, config.Area)
	for _, area := range areas {
		neighbors := neighborsMap[int(area.Id)]
		go podgen.applyWorker(area, neighbors)
		//defer podgen.deleteWorker(id) // not working...
	}

	// send area info to visualization
	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_VISUALIZATION,
	})
	logger.Debug("Send Area Info to Vis! \n%v\n", targets)
	//areas := []*api.Area{proc.Area}
	sclient := sclientOpts[uint32(api.ChannelType_AREA)].Sclient
	simapi.SendAreaInfoRequest(sclient, targets, areas)

	return true, nil
}

// startClock:
func (proc *Processor) startClock() {
	t1 := time.Now()

	targets := pm.GetProviderIds([]api.Provider_Type{
		api.Provider_WORKER,
		api.Provider_VISUALIZATION,
	})
	logger.Debug("Next Cycle! \n%v\n", targets)
	sclient := sclientOpts[uint32(api.ChannelType_CLOCK)].Sclient
	simapi.ForwardClockRequest(sclient, targets)

	// calc next time
	masterClock++
	log.Printf("\x1b[30m\x1b[47m \n Finish: Clock forwarded \n Time:  %v \x1b[0m\n", masterClock)

	t2 := time.Now()
	duration := t2.Sub(t1).Milliseconds()
	logger.Info("Duration: %v", duration)
	interval := int64(1000) // 周期ms
	if duration > interval {
		logger.Error("time cycle delayed...")
	} else {
		// 待機
		logger.Info("wait %v ms", interval-duration)
		time.Sleep(time.Duration(interval-duration) * time.Millisecond)
	}

	// 次のサイクルを行う
	if startFlag {
		proc.startClock()
	} else {
		log.Printf("\x1b[30m\x1b[47m \n Finish: Clock stopped \n GlobalTime:  %v \x1b[0m\n", masterClock)
		startFlag = false
		return
	}

}

// areaをrow、columnに分割する関数
func (proc *Processor) divideArea(areaCoords []*api.Coord, areaConfig Config_Area) ([]*api.Area, map[int][]string) {
	row := areaConfig.DefaultAreaNum.Row
	column := areaConfig.DefaultAreaNum.Column
	dupRange := areaConfig.DuplicateRange
	areas := []*api.Area{}
	neighborMap := make(map[int][]string)

	maxLat, maxLon, minLat, minLon := GetCoordRange(proc.Area.ControlArea)
	//areaId := 0
	for c := 0; c < int(column); c++ {
		// calc slon, elon
		slon := minLon + (maxLon-minLon)*float64(c)/float64(column)
		elon := minLon + (maxLon-minLon)*float64((c+1))/float64(column)
		for r := 0; r < int(row); r++ {
			//areaId := strconv.Itoa(c+1) + strconv.Itoa(r+1)
			areaIdint, _ := strconv.Atoi(strconv.Itoa(c+1) + strconv.Itoa(r+1))
			// calc slat, elat
			slat := minLat + (maxLat-minLat)*float64(r)/float64(row)
			elat := minLat + (maxLat-minLat)*float64((r+1))/float64(row)
			//fmt.Printf("test id %v\n", areaId)
			areas = append(areas, &api.Area{
				Id: uint64(areaIdint),
				ControlArea: []*api.Coord{
					{Latitude: slat, Longitude: slon},
					{Latitude: slat, Longitude: elon},
					{Latitude: elat, Longitude: elon},
					{Latitude: elat, Longitude: slon},
				},
				DuplicateArea: []*api.Coord{
					{Latitude: slat - dupRange, Longitude: slon - dupRange},
					{Latitude: slat - dupRange, Longitude: elon + dupRange},
					{Latitude: elat + dupRange, Longitude: elon + dupRange},
					{Latitude: elat + dupRange, Longitude: slon - dupRange},
				},
			})

			// add neighbors 各エリアの右と上を作成すれば全体を満たす
			if c+2 <= int(column) {
				id := strconv.Itoa(c+2) + strconv.Itoa(r+1)
				neighborMap[areaIdint] = append(neighborMap[areaIdint], id)
			}
			if r+2 <= int(row) {
				id := strconv.Itoa(c+1) + strconv.Itoa(r+2)
				neighborMap[areaIdint] = append(neighborMap[areaIdint], id)
			}
		}
	}

	return areas, neighborMap
}

///////////////////////////////////////////////
////////////  Order  //////////////////////
///////////////////////////////////////////////

type Order struct {
}

func NewOrder() *Order {
	order := &Order{}
	return order
}

type ClockOptions struct {
	Time int `validate:"required,min=0" json:"time"`
}

func (or *Order) SetClock() echo.HandlerFunc {
	return func(c echo.Context) error {
		co := new(ClockOptions)
		if err := c.Bind(co); err != nil {
			return err
		}
		fmt.Printf("time %d\n", co.Time)
		masterClock = co.Time
		return c.String(http.StatusOK, "Set Clock")
	}
}

type AgentOptions struct {
	Num int `validate:"required,min=0,max=10", json:"num"`
}

func (or *Order) SetAgent() echo.HandlerFunc {
	return func(c echo.Context) error {
		ao := new(AgentOptions)
		if err := c.Bind(ao); err != nil {
			return err
		}
		fmt.Printf("agent num %d\n", ao.Num)
		ok, err := proc.setAgents2(uint64(ao.Num))
		fmt.Printf("ok %v, err %v", ok, err)
		return c.String(http.StatusOK, "Set Agent")
	}
}

type AreaOptions struct {
	SLat string `min=0,max=100", json:"slat"`
	SLon string `min=0,max=200", json:"slon"`
	ELat string `min=0,max=100", json:"elat"`
	ELon string `min=0,max=200", json:"elon"`
}

func (or *Order) SetArea() echo.HandlerFunc {
	return func(c echo.Context) error {
		ao := new(AreaOptions)
		if err := c.Bind(ao); err != nil {
			return err
		}
		fmt.Printf("area %d\n", ao)
		slat, _ := strconv.ParseFloat(ao.SLat, 64)
		slon, _ := strconv.ParseFloat(ao.SLon, 64)
		elat, _ := strconv.ParseFloat(ao.ELat, 64)
		elon, _ := strconv.ParseFloat(ao.ELon, 64)
		area := []*api.Coord{
			{Latitude: slat, Longitude: slon},
			{Latitude: slat, Longitude: elon},
			{Latitude: elat, Longitude: elon},
			{Latitude: elat, Longitude: slon},
		}
		proc.setAreas(area)
		return c.String(http.StatusOK, "Set Area")
	}
}

func (or *Order) Start() echo.HandlerFunc {
	return func(c echo.Context) error {
		if startFlag == false {
			startFlag = true
			go proc.startClock()
			return c.String(http.StatusOK, "Start")
		} else {
			logger.Warn("Clock is already started.")
			return c.String(http.StatusBadRequest, "Start")
		}
	}
}

func (or *Order) Stop() echo.HandlerFunc {
	return func(c echo.Context) error {
		startFlag = false
		return c.String(http.StatusOK, "Stop")
	}
}

func startSimulatorServer() {
	fmt.Printf("Starting Simulator Server...")
	order := NewOrder()

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.POST("/order/set/clock", order.SetClock())
	e.POST("/order/set/agent", order.SetAgent())
	e.POST("/order/set/area", order.SetArea())
	e.POST("/order/start", order.Start())
	e.POST("/order/stop", order.Stop())

	e.Start(":" + port)
}

func main() {
	fmt.Printf("NumCPU=%d\n", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Printf("Start Main Provider")

	// CLI, GUIの受信サーバ
	go startSimulatorServer()

	// Register Node Server
	nodesrv := "127.0.0.1:9990"
	channelTypes := []uint32{}
	for _, opt := range sclientOpts {
		channelTypes = append(channelTypes, opt.ChType)
	}
	util.RegisterNodeLoop(nodesrv, "MasterProvider", channelTypes)

	// Register Synerex Server
	sxServerAddress := "127.0.0.1:10000"
	client := util.RegisterSynerexLoop(sxServerAddress)
	util.RegisterSXServiceClients(client, sclientOpts)

	wg := sync.WaitGroup{} // for syncing other goroutines
	wg.Add(1)
	wg.Wait()
	sxutil.CallDeferFunctions() // cleanup!

}

//////////////////////////////////
////////// Pod Generator ////////
//////////////////////////////////

type PodGenerator struct {
	RsrcMap map[string][]Resource
}

func NewPodGenerator() *PodGenerator {
	pg := &PodGenerator{
		RsrcMap: make(map[string][]Resource),
	}
	return pg
}

func (pg *PodGenerator) applyWorker(area *api.Area, neighbors []string) error {
	fmt.Printf("applying WorkerPod... %v\n", area.Id)
	areaid := strconv.FormatUint(area.Id, 10)
	rsrcs := []Resource{
		pg.NewWorkerService(areaid),
		pg.NewWorker(areaid),
		pg.NewAgent(areaid, area),
	}
	for _, neiId := range neighbors {
		rsrcs = append(rsrcs, pg.NewGateway(areaid, neiId))
	}
	fmt.Printf("applying WorkerPod2... %v\n", areaid)
	// write yaml
	fileName := "scripts/worker" + areaid + ".yaml"
	for _, rsrc := range rsrcs {
		err := WriteOnFile(fileName, rsrc)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	fmt.Printf("test: %v %v\n", fileName, areaid)
	// apply yaml
	cmd := exec.Command("kubectl", "apply", "-f", fileName)
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("Command Start Error. %v\n", err)
		return err
	}

	// delete yaml
	/*if err := os.Remove(fileName); err != nil {
		fmt.Println(err)
		return err
	}*/
	fmt.Printf("out: %v\n", string(out))

	// regist resource
	pg.RsrcMap[areaid] = rsrcs

	return nil
}

func (pg *PodGenerator) deleteWorker(areaid string) error {
	fmt.Printf("deleting WorkerPod...")
	rsrcs := pg.RsrcMap[areaid]

	// write yaml
	fileName := "worker" + areaid + ".yaml"
	for _, rsrc := range rsrcs {
		err := WriteOnFile(fileName, rsrc)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	// apply yaml
	cmd := exec.Command("kubectl", "delete", "-f", fileName)
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("Command Start Error.")
		return err
	}

	// delete yaml
	if err := os.Remove(fileName); err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Printf("out: %v\n", string(out))

	// regist resource
	pg.RsrcMap[areaid] = nil

	return nil
}

// gateway
func (pg *PodGenerator) NewGateway(areaId string, neiId string) Resource {
	worker1Name := "worker" + areaId
	worker2Name := "worker" + neiId
	gatewayName := "gateway" + areaId + neiId
	gateway := Resource{
		ApiVersion: "v1",
		Kind:       "Pod",
		Metadata: Metadata{
			Name:   gatewayName,
			Labels: Label{App: gatewayName},
		},
		Spec: Spec{
			Containers: []Container{
				{
					Name:            "gateway-provider",
					Image:           "synerex-simulation/gateway-provider:latest",
					ImagePullPolicy: "Never",
					Env: []Env{
						{
							Name:  "WORKER_SYNEREX_SERVER1",
							Value: worker1Name + ":700",
						},
						{
							Name:  "WORKER_NODEID_SERVER1",
							Value: worker1Name + ":600",
						},
						{
							Name:  "WORKER_SYNEREX_SERVER2",
							Value: worker2Name + ":700",
						},
						{
							Name:  "WORKER_NODEID_SERVER2",
							Value: worker2Name + ":600",
						},
						{
							Name:  "PROVIDER_NAME",
							Value: "GatewayProvider" + areaId + neiId,
						},
					},
					Ports: []Port{{ContainerPort: 9980}},
				},
			},
		},
	}
	return gateway
}

func (pg *PodGenerator) NewAgent(areaid string, area *api.Area) Resource {
	workerName := "worker" + areaid
	agentName := "agent" + areaid
	agent := Resource{
		ApiVersion: "v1",
		Kind:       "Pod",
		Metadata: Metadata{
			Name:   agentName,
			Labels: Label{App: agentName},
		},
		Spec: Spec{
			Containers: []Container{
				{
					Name:            "agent-provider",
					Image:           "synerex-simulation/agent-provider:latest",
					ImagePullPolicy: "Never",
					Env: []Env{
						{
							Name:  "NODEID_SERVER",
							Value: workerName + ":600",
						},
						{
							Name:  "SYNEREX_SERVER",
							Value: workerName + ":700",
						},
						{
							Name:  "VIS_SYNEREX_SERVER",
							Value: "visualization:700",
						},
						{
							Name:  "VIS_NODEID_SERVER",
							Value: "visualization:600",
						},
						{
							Name:  "AREA",
							Value: convertAreaToJson(area),
						},
						{
							Name:  "PROVIDER_NAME",
							Value: "AgentProvider" + areaid,
						},
					},
				},
			},
		},
	}
	return agent
}

// worker
func (pg *PodGenerator) NewWorkerService(areaid string) Resource {
	name := "worker" + areaid
	service := Resource{
		ApiVersion: "v1",
		Kind:       "Service",
		Metadata:   Metadata{Name: name},
		Spec: Spec{
			Selector: Selector{App: name},
			Ports: []Port{
				{
					Name:       "synerex",
					Port:       700,
					TargetPort: 10000,
				},
				{
					Name:       "nodeid",
					Port:       600,
					TargetPort: 9000,
				},
			},
		},
	}
	return service
}

func (pg *PodGenerator) NewWorker(areaid string) Resource {
	name := "worker" + areaid
	worker := Resource{
		ApiVersion: "v1",
		Kind:       "Pod",
		Metadata: Metadata{
			Name:   name,
			Labels: Label{App: name},
		},
		Spec: Spec{
			Containers: []Container{
				{
					Name:            "nodeid-server",
					Image:           "synerex-simulation/nodeid-server:latest",
					ImagePullPolicy: "Never",
					Env: []Env{
						{
							Name:  "NODEID_SERVER",
							Value: ":9000",
						},
					},
					Ports: []Port{{ContainerPort: 9000}},
				},
				{
					Name:            "synerex-server",
					Image:           "synerex-simulation/synerex-server:latest",
					ImagePullPolicy: "Never",
					Env: []Env{
						{
							Name:  "NODEID_SERVER",
							Value: ":9000",
						},
						{
							Name:  "SYNEREX_SERVER",
							Value: ":10000",
						},
						{
							Name:  "SERVER_NAME",
							Value: "SynerexServer" + areaid,
						},
					},
					Ports: []Port{{ContainerPort: 10000}},
				},
				{
					Name:            "worker-provider",
					Image:           "synerex-simulation/worker-provider:latest",
					ImagePullPolicy: "Never",
					Env: []Env{
						{
							Name:  "NODEID_SERVER",
							Value: ":9000",
						},
						{
							Name:  "SYNEREX_SERVER",
							Value: ":10000",
						},
						{
							Name:  "MASTER_SYNEREX_SERVER",
							Value: "master:700",
						},
						{
							Name:  "MASTER_NODEID_SERVER",
							Value: "master:600",
						},
						{
							Name:  "PORT",
							Value: "9980",
						},
						{
							Name:  "PROVIDER_NAME",
							Value: "WorkerProvider" + areaid,
						},
					},
					Ports: []Port{{ContainerPort: 9980}},
				},
			},
		},
	}
	return worker
}

// ファイル名とデータをを渡すとyamlファイルに保存してくれる関数です。
func WriteOnFile(fileName string, data interface{}) error {
	// ここでデータを []byte に変換しています。
	buf, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		//エラー処理
		log.Fatal(err)
	}
	defer file.Close()
	fmt.Fprintln(file, string(buf))   //書き込み
	fmt.Fprintln(file, string("---")) //書き込み
	return nil
}

func convertAreaToJson(area *api.Area) string {
	id := area.Id
	duplicateText := `[`
	controlText := `[`
	for i, ctl := range area.ControlArea {
		ctlText := fmt.Sprintf(`{"latitude":%v, "longitude":%v}`, ctl.Latitude, ctl.Longitude)
		//fmt.Printf("ctl %v\n", ctlText)
		if i == len(area.ControlArea)-1 { // 最後は,をつけない
			controlText += ctlText
		} else {
			controlText += ctlText + ","
		}
	}
	for i, dpl := range area.DuplicateArea {
		dplText := fmt.Sprintf(`{"latitude":%v, "longitude":%v}`, dpl.Latitude, dpl.Longitude)
		//fmt.Printf("dpl %v\n", dplText)
		if i == len(area.DuplicateArea)-1 { // 最後は,をつけない
			duplicateText += dplText
		} else {
			duplicateText += dplText + ","
		}
	}

	duplicateText += `]`
	controlText += `]`
	result := fmt.Sprintf(`{"id":%d, "name":"Unknown", "duplicate_area": %s, "control_area": %s}`, id, duplicateText, controlText)
	//result = fmt.Sprintf("%s", result)
	//fmt.Printf("areaJson: %s\n", result)
	return result
}

type Resource struct {
	ApiVersion string   `yaml:"apiVersion,omitempty"`
	Kind       string   `yaml:"kind,omitempty"`
	Metadata   Metadata `yaml:"metadata,omitempty"`
	Spec       Spec     `yaml:"spec,omitempty"`
}

type Spec struct {
	Containers []Container `yaml:"containers,omitempty"`
	Selector   Selector    `yaml:"selector,omitempty"`
	Ports      []Port      `yaml:"ports,omitempty"`
	Type       string      `yaml:"type,omitempty"`
}

type Container struct {
	Name            string `yaml:"name,omitempty"`
	Image           string `yaml:"image,omitempty"`
	ImagePullPolicy string `yaml:"imagePullPolicy,omitempty"`
	Stdin           bool   `yaml:"stdin,omitempty"`
	Tty             bool   `yaml:"tty,omitempty"`
	Env             []Env  `yaml:"env,omitempty"`
	Ports           []Port `yaml:"ports,omitempty"`
}

type Env struct {
	Name  string `yaml:"name,omitempty"`
	Value string `yaml:"value,omitempty"`
}

type Selector struct {
	App         string `yaml:"app,omitempty"`
	MatchLabels Label  `yaml:"matchLabels,omitempty"`
}

type Port struct {
	Name          string `yaml:"name,omitempty"`
	Port          int    `yaml:"port,omitempty"`
	TargetPort    int    `yaml:"targetPort,omitempty"`
	ContainerPort int    `yaml:"containerPort,omitempty"`
}

type Metadata struct {
	Name   string `yaml:"name,omitempty"`
	Labels Label  `yaml:"labels,omitempty"`
}

type Label struct {
	App string `yaml:"app,omitempty"`
}

type Area struct {
	Id        int
	Control   []*api.Coord
	Duplicate []*api.Coord
}

type Coord struct {
	Latitude  float64
	Longitude float64
}

func GetCoordRange(coords []*api.Coord) (float64, float64, float64, float64) {
	maxLon, maxLat := math.Inf(-1), math.Inf(-1)
	minLon, minLat := math.Inf(0), math.Inf(0)
	for _, coord := range coords {
		if coord.Latitude > maxLat {
			maxLat = coord.Latitude
		}
		if coord.Longitude > maxLon {
			maxLon = coord.Longitude
		}
		if coord.Latitude < minLat {
			minLat = coord.Latitude
		}
		if coord.Longitude < minLon {
			minLon = coord.Longitude
		}
	}
	return maxLat, maxLon, minLat, minLon
}

/////////////////////////////////////////////////////
//////// util for creating higashiyama route ////////
///////////////////////////////////////////////////////

type RoutePoint struct {
	Id             uint64
	Name           string
	Point          *api.Coord
	NeighborPoints []*RoutePoint
}

func GetRoutes() []*RoutePoint {
	routes := []*RoutePoint{
		{
			Id: 0, Name: "gate", Point: &api.Coord{Longitude: 136.974024, Latitude: 35.158995},
			NeighborPoints: []*RoutePoint{
				{Id: 1, Name: "enterance", Point: &api.Coord{Longitude: 136.974688, Latitude: 35.158228}},
			},
		},
		{
			Id: 1, Name: "enterance", Point: &api.Coord{Longitude: 136.974688, Latitude: 35.158228},
			NeighborPoints: []*RoutePoint{
				{Id: 0, Name: "gate", Point: &api.Coord{Longitude: 136.974024, Latitude: 35.158995}},
				{Id: 2, Name: "rightEnt", Point: &api.Coord{Longitude: 136.974645, Latitude: 35.157958}},
				{Id: 3, Name: "leftEnt", Point: &api.Coord{Longitude: 136.974938, Latitude: 35.158164}},
			},
		},
		{
			Id: 2, Name: "rightEnt", Point: &api.Coord{Longitude: 136.974645, Latitude: 35.157958},
			NeighborPoints: []*RoutePoint{
				{Id: 1, Name: "enterance", Point: &api.Coord{Longitude: 136.974688, Latitude: 35.158228}},
				{Id: 4, Name: "road1", Point: &api.Coord{Longitude: 136.974864, Latitude: 35.157823}},
			},
		},
		{
			Id: 3, Name: "leftEnt", Point: &api.Coord{Longitude: 136.974938, Latitude: 35.158164},
			NeighborPoints: []*RoutePoint{
				{Id: 1, Name: "enterance", Point: &api.Coord{Longitude: 136.974688, Latitude: 35.158228}},
				{Id: 5, Name: "road2", Point: &api.Coord{Longitude: 136.975054, Latitude: 35.158001}},
				{Id: 17, Name: "north1", Point: &api.Coord{Longitude: 136.976395, Latitude: 35.158410}},
			},
		},
		{
			Id: 4, Name: "road1", Point: &api.Coord{Longitude: 136.974864, Latitude: 35.157823},
			NeighborPoints: []*RoutePoint{
				{Id: 2, Name: "rightEnt", Point: &api.Coord{Longitude: 136.974645, Latitude: 35.157958}},
				{Id: 5, Name: "road2", Point: &api.Coord{Longitude: 136.975054, Latitude: 35.158001}},
				{Id: 6, Name: "road3", Point: &api.Coord{Longitude: 136.975517, Latitude: 35.157096}},
			},
		},
		{
			Id: 5, Name: "road2", Point: &api.Coord{Longitude: 136.975054, Latitude: 35.158001},
			NeighborPoints: []*RoutePoint{
				{Id: 3, Name: "leftEnt", Point: &api.Coord{Longitude: 136.974938, Latitude: 35.158164}},
				{Id: 4, Name: "road1", Point: &api.Coord{Longitude: 136.974864, Latitude: 35.157823}},
			},
		},
		{
			Id: 6, Name: "road3", Point: &api.Coord{Longitude: 136.975517, Latitude: 35.157096},
			NeighborPoints: []*RoutePoint{
				{Id: 7, Name: "road4", Point: &api.Coord{Longitude: 136.975872, Latitude: 35.156678}},
				{Id: 4, Name: "road1", Point: &api.Coord{Longitude: 136.974864, Latitude: 35.157823}},
			},
		},
		{
			Id: 7, Name: "road4", Point: &api.Coord{Longitude: 136.975872, Latitude: 35.156678},
			NeighborPoints: []*RoutePoint{
				{Id: 6, Name: "road3", Point: &api.Coord{Longitude: 136.975517, Latitude: 35.157096}},
				{Id: 8, Name: "road5", Point: &api.Coord{Longitude: 136.976314, Latitude: 35.156757}},
				{Id: 10, Name: "burger", Point: &api.Coord{Longitude: 136.976960, Latitude: 35.155697}},
			},
		},
		{
			Id: 8, Name: "road5", Point: &api.Coord{Longitude: 136.976314, Latitude: 35.156757},
			NeighborPoints: []*RoutePoint{
				{Id: 6, Name: "road3", Point: &api.Coord{Longitude: 136.975517, Latitude: 35.157096}},
				{Id: 9, Name: "toilet", Point: &api.Coord{Longitude: 136.977261, Latitude: 35.155951}},
			},
		},
		{
			Id: 9, Name: "toilet", Point: &api.Coord{Longitude: 136.977261, Latitude: 35.155951},
			NeighborPoints: []*RoutePoint{
				{Id: 8, Name: "road5", Point: &api.Coord{Longitude: 136.976314, Latitude: 35.156757}},
				{Id: 10, Name: "burger", Point: &api.Coord{Longitude: 136.976960, Latitude: 35.155697}},
			},
		},
		{
			Id: 10, Name: "burger", Point: &api.Coord{Longitude: 136.976960, Latitude: 35.155697},
			NeighborPoints: []*RoutePoint{
				{Id: 8, Name: "road5", Point: &api.Coord{Longitude: 136.976314, Latitude: 35.156757}},
				{Id: 7, Name: "road4", Point: &api.Coord{Longitude: 136.975872, Latitude: 35.156678}},
				{Id: 11, Name: "lake1", Point: &api.Coord{Longitude: 136.978217, Latitude: 35.155266}},
			},
		},
		{
			Id: 11, Name: "lake1", Point: &api.Coord{Longitude: 136.978217, Latitude: 35.155266},
			NeighborPoints: []*RoutePoint{
				{Id: 10, Name: "burger", Point: &api.Coord{Longitude: 136.976960, Latitude: 35.155697}},
				{Id: 12, Name: "lake2", Point: &api.Coord{Longitude: 136.978623, Latitude: 35.155855}},
				{Id: 16, Name: "lake6", Point: &api.Coord{Longitude: 136.978297, Latitude: 35.154755}},
			},
		},
		{
			Id: 12, Name: "lake2", Point: &api.Coord{Longitude: 136.978623, Latitude: 35.155855},
			NeighborPoints: []*RoutePoint{
				{Id: 11, Name: "lake1", Point: &api.Coord{Longitude: 136.978217, Latitude: 35.155266}},
				{Id: 13, Name: "lake3", Point: &api.Coord{Longitude: 136.979657, Latitude: 35.155659}},
			},
		},
		{
			Id: 13, Name: "lake3", Point: &api.Coord{Longitude: 136.979657, Latitude: 35.155659},
			NeighborPoints: []*RoutePoint{
				{Id: 12, Name: "lake2", Point: &api.Coord{Longitude: 136.978623, Latitude: 35.155855}},
				{Id: 14, Name: "lake4", Point: &api.Coord{Longitude: 136.980489, Latitude: 35.154484}},
				{Id: 26, Name: "east6", Point: &api.Coord{Longitude: 136.984100, Latitude: 35.153693}},
				{Id: 22, Name: "east1", Point: &api.Coord{Longitude: 136.981124, Latitude: 35.157283}},
				{Id: 27, Name: "east-in1", Point: &api.Coord{Longitude: 136.982804, Latitude: 35.154175}},
			},
		},
		{
			Id: 14, Name: "lake4", Point: &api.Coord{Longitude: 136.980489, Latitude: 35.154484},
			NeighborPoints: []*RoutePoint{
				{Id: 13, Name: "lake3", Point: &api.Coord{Longitude: 136.979657, Latitude: 35.155659}},
				{Id: 15, Name: "lake5", Point: &api.Coord{Longitude: 136.980143, Latitude: 35.153869}},
			},
		},
		{
			Id: 15, Name: "lake5", Point: &api.Coord{Longitude: 136.980143, Latitude: 35.153869},
			NeighborPoints: []*RoutePoint{
				{Id: 14, Name: "lake4", Point: &api.Coord{Longitude: 136.980489, Latitude: 35.154484}},
				{Id: 16, Name: "lake6", Point: &api.Coord{Longitude: 136.978297, Latitude: 35.154755}},
			},
		},
		{
			Id: 16, Name: "lake6", Point: &api.Coord{Longitude: 136.978297, Latitude: 35.154755},
			NeighborPoints: []*RoutePoint{
				{Id: 11, Name: "lake1", Point: &api.Coord{Longitude: 136.978217, Latitude: 35.155266}},
				{Id: 15, Name: "lake5", Point: &api.Coord{Longitude: 136.980143, Latitude: 35.153869}},
			},
		},
		{
			Id: 17, Name: "north1", Point: &api.Coord{Longitude: 136.976395, Latitude: 35.158410},
			NeighborPoints: []*RoutePoint{
				{Id: 3, Name: "leftEnt", Point: &api.Coord{Longitude: 136.974938, Latitude: 35.158164}},
				{Id: 5, Name: "road2", Point: &api.Coord{Longitude: 136.975054, Latitude: 35.158001}},
				{Id: 18, Name: "north2", Point: &api.Coord{Longitude: 136.977821, Latitude: 35.159220}},
			},
		},
		{
			Id: 18, Name: "north2", Point: &api.Coord{Longitude: 136.977821, Latitude: 35.159220},
			NeighborPoints: []*RoutePoint{
				{Id: 17, Name: "north1", Point: &api.Coord{Longitude: 136.976395, Latitude: 35.158410}},
				{Id: 19, Name: "medaka", Point: &api.Coord{Longitude: 136.979040, Latitude: 35.158147}},
			},
		},
		{
			Id: 19, Name: "medaka", Point: &api.Coord{Longitude: 136.979040, Latitude: 35.158147},
			NeighborPoints: []*RoutePoint{
				{Id: 18, Name: "north2", Point: &api.Coord{Longitude: 136.977821, Latitude: 35.159220}},
				{Id: 20, Name: "tower", Point: &api.Coord{Longitude: 136.978846, Latitude: 35.157108}},
			},
		},
		{
			Id: 20, Name: "tower", Point: &api.Coord{Longitude: 136.978846, Latitude: 35.157108},
			NeighborPoints: []*RoutePoint{
				{Id: 19, Name: "medaka", Point: &api.Coord{Longitude: 136.979040, Latitude: 35.158147}},
				{Id: 21, Name: "north-out", Point: &api.Coord{Longitude: 136.977890, Latitude: 35.156563}},
			},
		},
		{
			Id: 21, Name: "north-out", Point: &api.Coord{Longitude: 136.977890, Latitude: 35.156563},
			NeighborPoints: []*RoutePoint{
				{Id: 20, Name: "tower", Point: &api.Coord{Longitude: 136.978846, Latitude: 35.157108}},
				{Id: 17, Name: "north1", Point: &api.Coord{Longitude: 136.976395, Latitude: 35.158410}},
				{Id: 9, Name: "toilet", Point: &api.Coord{Longitude: 136.977261, Latitude: 35.155951}},
			},
		},
		{
			Id: 22, Name: "east1", Point: &api.Coord{Longitude: 136.981124, Latitude: 35.157283},
			NeighborPoints: []*RoutePoint{
				{Id: 13, Name: "lake3", Point: &api.Coord{Longitude: 136.979657, Latitude: 35.155659}},
				{Id: 23, Name: "east2", Point: &api.Coord{Longitude: 136.984350, Latitude: 35.157271}},
			},
		},
		{
			Id: 23, Name: "east2", Point: &api.Coord{Longitude: 136.984350, Latitude: 35.157271},
			NeighborPoints: []*RoutePoint{
				{Id: 22, Name: "east1", Point: &api.Coord{Longitude: 136.981124, Latitude: 35.157283}},
				{Id: 24, Name: "east3", Point: &api.Coord{Longitude: 136.987567, Latitude: 35.158233}},
			},
		},
		{
			Id: 24, Name: "east3", Point: &api.Coord{Longitude: 136.987567, Latitude: 35.158233},
			NeighborPoints: []*RoutePoint{
				{Id: 23, Name: "east2", Point: &api.Coord{Longitude: 136.984350, Latitude: 35.157271}},
				{Id: 25, Name: "east4", Point: &api.Coord{Longitude: 136.988522, Latitude: 35.157286}},
			},
		},
		{
			Id: 25, Name: "east4", Point: &api.Coord{Longitude: 136.988522, Latitude: 35.157286},
			NeighborPoints: []*RoutePoint{
				{Id: 24, Name: "east3", Point: &api.Coord{Longitude: 136.987567, Latitude: 35.158233}},
				{Id: 25, Name: "east5", Point: &api.Coord{Longitude: 136.988355, Latitude: 35.155838}},
			},
		},
		{
			Id: 25, Name: "east5", Point: &api.Coord{Longitude: 136.988355, Latitude: 35.155838},
			NeighborPoints: []*RoutePoint{
				{Id: 25, Name: "east4", Point: &api.Coord{Longitude: 136.988522, Latitude: 35.157286}},
				{Id: 26, Name: "east6", Point: &api.Coord{Longitude: 136.984100, Latitude: 35.153693}},
			},
		},
		{
			Id: 26, Name: "east6", Point: &api.Coord{Longitude: 136.984100, Latitude: 35.153693},
			NeighborPoints: []*RoutePoint{
				{Id: 25, Name: "east5", Point: &api.Coord{Longitude: 136.988355, Latitude: 35.155838}},
				{Id: 13, Name: "lake3", Point: &api.Coord{Longitude: 136.979657, Latitude: 35.155659}},
				{Id: 27, Name: "east-in1", Point: &api.Coord{Longitude: 136.982804, Latitude: 35.154175}},
			},
		},
		{
			Id: 27, Name: "east-in1", Point: &api.Coord{Longitude: 136.982804, Latitude: 35.154175},
			NeighborPoints: []*RoutePoint{
				{Id: 26, Name: "east6", Point: &api.Coord{Longitude: 136.984100, Latitude: 35.153693}},
				{Id: 13, Name: "lake3", Point: &api.Coord{Longitude: 136.979657, Latitude: 35.155659}},
				{Id: 28, Name: "east-in2", Point: &api.Coord{Longitude: 136.984244, Latitude: 35.156283}},
			},
		},
		{
			Id: 28, Name: "east-in2", Point: &api.Coord{Longitude: 136.984244, Latitude: 35.156283},
			NeighborPoints: []*RoutePoint{
				{Id: 29, Name: "east-in3", Point: &api.Coord{Longitude: 136.987627, Latitude: 35.157104}},
				{Id: 27, Name: "east-in1", Point: &api.Coord{Longitude: 136.982804, Latitude: 35.154175}},
			},
		},
		{
			Id: 29, Name: "east-in3", Point: &api.Coord{Longitude: 136.987627, Latitude: 35.157104},
			NeighborPoints: []*RoutePoint{
				{Id: 28, Name: "east-in2", Point: &api.Coord{Longitude: 136.984244, Latitude: 35.156283}},
				{Id: 30, Name: "east-in4", Point: &api.Coord{Longitude: 136.986063, Latitude: 35.155353}},
			},
		},
		{
			Id: 30, Name: "east-in4", Point: &api.Coord{Longitude: 136.986063, Latitude: 35.155353},
			NeighborPoints: []*RoutePoint{
				{Id: 29, Name: "east-in3", Point: &api.Coord{Longitude: 136.987627, Latitude: 35.157104}},
				{Id: 26, Name: "east6", Point: &api.Coord{Longitude: 136.984100, Latitude: 35.153693}},
			},
		},
	}

	return routes
}
func GetAmongPosition(pos1 *api.Coord, pos2 *api.Coord) *api.Coord {
	lat1 := pos1.Latitude
	lon1 := pos1.Longitude
	lat2 := pos2.Latitude
	lon2 := pos2.Longitude
	position := &api.Coord{
		Latitude:  lat1 + (lat2-lat1)*rand.Float64(),
		Longitude: lon1 + (lon2-lon1)*rand.Float64(),
	}
	return position
}
