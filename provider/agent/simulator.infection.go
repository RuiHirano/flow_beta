package main

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"strconv"
	"strings"

	"github.com/jszwec/csvutil"

	"math/rand"

	api "github.com/RuiHirano/flow_beta/api"
	algo "github.com/RuiHirano/flow_beta/provider/agent/algorithm"
	"github.com/google/uuid"
)

var (
	infection *algo.Infection
	linkData map[string]Link
	nodeData map[string]Node
)
func init(){	

	linkData = getLinkData()
	nodeData = getNodeData()
	param := &algo.ModelParam{
		Radius: 0.00006,  // 2m
		Rate: 0.8,  // 80%
		DefaultInfRate: 0.2, // 20%
	}
	infection = algo.NewInfection(param)
}

// SynerexSimulator :
type Simulator struct {
	DiffAgents []*api.Agent //　同じエリアの異種エージェント
	Agents     []*api.Agent
	Area       *api.Area
	AgentType  api.AgentType
}

// NewSenerexSimulator:
func NewSimulator(areaInfo *api.Area, agentType api.AgentType) *Simulator {

	sim := &Simulator{
		DiffAgents: make([]*api.Agent, 0),
		Agents:     make([]*api.Agent, 0),
		Area:       areaInfo,
		AgentType:  agentType,
	}

	return sim
}

// AddAgents :　Agentsを追加する関数
func (sim *Simulator) AddAgents(agents []*api.Agent) int {

	infAgents := CreateInfectionAgents(agents)
	
	newAgents := make([]*api.Agent, 0)
	for _, infAgent := range infAgents {
		if infAgent.Type == sim.AgentType {
			position := infAgent.Route.Position
			//("Debug %v, %v", position, sim.Area.DuplicateArea)
			if IsAgentInArea(position, sim.Area.DuplicateArea) {
				newAgents = append(newAgents, infAgent)
			}
		}
	}
	sim.Agents = append(sim.Agents, newAgents...)

	return len(sim.Agents)
}

// SetAgents :　Agentsをセットする関数
func (sim *Simulator) SetDiffAgents(agentsInfo []*api.Agent) {
	newAgents := make([]*api.Agent, 0)
	for _, agentInfo := range agentsInfo {
		if agentInfo.Type == sim.AgentType {
			position := agentInfo.Route.Position
			if IsAgentInArea(position, sim.Area.DuplicateArea) {
				newAgents = append(newAgents, agentInfo)
			}
		}
	}
	sim.DiffAgents = newAgents
}

// SetAgents :　Agentsをセットする関数
func (sim *Simulator) SetAgents(agentsInfo []*api.Agent) {
	newAgents := make([]*api.Agent, 0)
	for _, agentInfo := range agentsInfo {
		if agentInfo.Type == sim.AgentType {
			position := agentInfo.Route.Position
			//("Debug %v, %v", position, sim.Area.DuplicateArea)
			if IsAgentInArea(position, sim.Area.DuplicateArea) {
				newAgents = append(newAgents, agentInfo)
			}
		}
	}
	sim.Agents = newAgents
}

// ResetAgents :　Agentsを追加する関数
func (sim *Simulator) ResetAgents() {
	sim.Agents = make([]*api.Agent, 0)
}

// SimulationRequest :　Agentsを追加する関数
func (sim *Simulator) SimulationRequest(simReq *api.SimulatorRequest) {
	infection.SimulationRequest(simReq)
}

// GetAgents :　Agentsを取得する関数
func (sim *Simulator) GetAgents() []*api.Agent {
	return sim.Agents
}

// UpdateDuplicateAgents :　重複エリアのエージェントを更新する関数
func (sim *Simulator) UpdateDuplicateAgents(neighborAgents []*api.Agent) []*api.Agent {
	nextAgents := sim.Agents
	for _, neighborAgent := range neighborAgents {
		isAppendAgent := true
		position := neighborAgent.Route.Position
		for _, sameAreaAgent := range sim.Agents {
			// 自分の管理しているエージェントではなく重複エリアに入っていた場合更新する
			//FIX Duplicateじゃない？
			if neighborAgent.Id == sameAreaAgent.Id {
				isAppendAgent = false
			}
		}
		if isAppendAgent && IsAgentInArea(position, sim.Area.DuplicateArea) {
			nextAgents = append(nextAgents, neighborAgent)
		}
	}
	sim.Agents = nextAgents
	return nextAgents
}

// ForwardStep :　次の時刻のエージェントを計算する関数
func (sim *Simulator) ForwardStep() []*api.Agent {

	//nextAgents := sim.GetAgents()
	// Agent計算
	//sameAgents := sim.DiffAgents
	//rvo2route := algo.NewRVO2Route(sim.Agents, sim.Area)
	/*param := &algo.ModelParam{
		Radius: 0.00006,  // 2m
		Rate: 0.8,  // 80%
	}
	infection := algo.NewInfection(param, sim.Agents, sim.Area)*/
	infection.SetAgents(sim.Agents)
	infection.SetArea(sim.Area)
	nextAgents := infection.CalcNextAgents()
	sim.Agents = nextAgents

	//simpleroute := algo.NewSimpleRoute2(sim.Agents)
	//nextAgents = simpleroute.CalcNextAgents()

	return nextAgents
}

// エージェントがエリアの中にいるかどうか
func IsAgentInArea(position *api.Coord, areaCoords []*api.Coord) bool {
	lat := position.Latitude
	lon := position.Longitude
	maxLat, maxLon, minLat, minLon := GetCoordRange(areaCoords)
	if minLat < lat && lat < maxLat && minLon < lon && lon < maxLon {
		return true
	} else {
		return false
	}
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

// for experiment
func CreateInfectionAgents(agents []*api.Agent)[]*api.Agent{
	expAgents := []*api.Agent{}
	//maxLat, maxLon, minLat, minLon := GetCoordRange(myArea.ControlArea)
	for range agents{

		uid, _ := uuid.NewRandom()
		departure, destination, transitPoints := createRoute(10, nodeData, linkData)
		/*position := &api.Coord{
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

		transitPoints := []*api.Coord{transitPoint}*/
		var agentParam  *algo.AgentParam
		if rand.Float64() < infection.ModelParam.DefaultInfRate{  // 5%以下が初期感染
			agentParam = &algo.AgentParam{
				Status: "I",
				Move: 0,
			}
		}else{
			agentParam = &algo.AgentParam{
			Status: "S",
			Move: 0,
			}
		}
		agentModelParamJson, _ := json.Marshal(agentParam)
		expAgents = append(expAgents, &api.Agent{
			Type: api.AgentType_PEDESTRIAN,
			Id:   uint64(uid.ID()),
			Route: &api.Route{
				Position:      departure,
				Direction:     30,
				Speed:         60,
				Departure:     departure,
				Destination:   destination,
				TransitPoints: transitPoints,
				NextTransit:   transitPoints[0],
			},
			Data: string(agentModelParamJson),
		})
	}
	return expAgents
}




///////////////////////////////////////////////
////////   Nagoya Route ///////////////////////
//////////////////////////////////////////////

type Link struct {
	LinkID string `csv:"LinkID"`
	StartNodeID string `csv:"StartNodeID"`
	EndNodeID string `csv:"EndNodeID"`
	Type int `csv:"Type"`
}

func getLinkData() map[string]Link{
	var linkData []Link
// バイト列を読み込む
	b, _ := ioutil.ReadFile("./data/nagoya/link.csv")
	// ユーザー定義型スライスにマッピング
	csvutil.Unmarshal(b, &linkData)

	linkMap := make(map[string]Link)
	for _, v := range linkData{
		linkMap[v.LinkID] = v
	}
	return linkMap
}

type Exit struct {
	ExitID string `csv:"ExitID"`
	NodeID string `csv:"NodeID"`
	FacilityID string `csv:"FacilityID"`
	Name string `csv:"Name"`
}

func getExitData() map[string]Exit{
	var exitData []Exit
// バイト列を読み込む
	b, _ := ioutil.ReadFile("./data/nagoya/exit.csv")
	// ユーザー定義型スライスにマッピング
	csvutil.Unmarshal(b, &exitData)

	exitMap := make(map[string]Exit)
	for _, v := range exitData{
		exitMap[v.ExitID] = v
	}
	return exitMap
}

func convertToCoord(wgs84 string) float64{
	values := strings.Split(wgs84, ".")
	degrees, _ := strconv.ParseFloat(values[0], 64)
	minutes, _ := strconv.ParseFloat(values[1], 64)
	seconds1, _ := strconv.ParseFloat(values[2], 64)
	var seconds float64
	if len(values) == 3{
		seconds = seconds1
	}else{
		seconds2, _ := strconv.ParseFloat(values[3], 64)
		seconds = seconds1 + seconds2/10000
	}
	result := degrees + minutes/60 +seconds/3600
	return result
}

type NodeCSV struct {
	NodeID string `csv:"NodeID"`
	Code int `csv:"Code"`
	Latitude string `csv:"Latitude"`
	Longitude string `csv:"Longitude"`
	Height float64 `csv:"Height"`
	LinkID1 string `csv:"LinkID1"`
	LinkID2 string `csv:"LinkID2"`
	LinkID3 string `csv:"LinkID3"`
	LinkID4 string `csv:"LinkID4"`
	LinkID5 string `csv:"LinkID5"`
	LinkID6 string `csv:"LinkID6"`
}

type Node struct {
	NodeID string `csv:"NodeID"`
	Code int `csv:"Code"`
	Latitude float64 `csv:"Latitude"`
	Longitude float64 `csv:"Longitude"`
	Height float64 `csv:"Height"`
	LinkID1 string `csv:"LinkID1"`
	LinkID2 string `csv:"LinkID2"`
	LinkID3 string `csv:"LinkID3"`
	LinkID4 string `csv:"LinkID4"`
	LinkID5 string `csv:"LinkID5"`
	LinkID6 string `csv:"LinkID6"`
}

func getNodeData() map[string]Node{
	var nodeData []Node
// バイト列を読み込む
	var nodeCSVData []NodeCSV
	b, _ := ioutil.ReadFile("./data/nagoya/node.csv")
	// ユーザー定義型スライスにマッピング
	csvutil.Unmarshal(b, &nodeCSVData)
	
	for _, v := range nodeCSVData{
		nodeData = append(nodeData, Node{
			NodeID : v.NodeID,
			Code : v.Code,
			Latitude : convertToCoord(v.Latitude),
			Longitude : convertToCoord(v.Longitude),
			Height : v.Height,
			LinkID1 : v.LinkID1,
			LinkID2 : v.LinkID2,
			LinkID3 : v.LinkID3,
			LinkID4 : v.LinkID4,
			LinkID5 : v.LinkID5,
			LinkID6 : v.LinkID6,
		})
	}

	nodeMap := make(map[string]Node)
	for _, v := range nodeData{
		nodeMap[v.NodeID] = v
	}
	return nodeMap
}

type FacilityCSV struct {
	FacilityID string `csv:"FacilityID"`
	Name string `csv:"Name"`
	Code int `csv:"Code"`
	Latitude string `csv:"Latitude"`
	Longitude string `csv:"Longitude"`
}

type Facility struct {
	FacilityID string `csv:"FacilityID"`
	Name string `csv:"Name"`
	Code int `csv:"Code"`
	Latitude float64 `csv:"Latitude"`
	Longitude float64 `csv:"Longitude"`
}

func getFacilityData() map[string]Facility{
	var facilityData []Facility
// バイト列を読み込む
	var facilityCSVData []FacilityCSV
	b, _ := ioutil.ReadFile("./data/nagoya/facility.csv")
	// ユーザー定義型スライスにマッピング
	csvutil.Unmarshal(b, &facilityCSVData)
	
	for _, v := range facilityCSVData{
		facilityData = append(facilityData, Facility{
			FacilityID : v.FacilityID,
			Name: v.Name,
			Code : v.Code,
			Latitude : convertToCoord(v.Latitude),
			Longitude : convertToCoord(v.Longitude),
		})
	}

	facilityMap := make(map[string]Facility)
	for _, v := range facilityData{
		facilityMap[v.FacilityID] = v
	}
	return facilityMap
}

type HospitalCSV struct {
	HospitalID string `csv:"HospitalID"`
	Name string `csv:"Name"`
	Code int `csv:"Code"`
	Latitude string `csv:"Latitude"`
	Longitude string `csv:"Longitude"`
}

type Hospital struct {
	HospitalID string `csv:"HospitalID"`
	Name string `csv:"Name"`
	Code int `csv:"Code"`
	Latitude float64 `csv:"Latitude"`
	Longitude float64 `csv:"Longitude"`
}

func getHospitalData() map[string]Hospital{
	var hospitalData []Hospital
// バイト列を読み込む
	var hospitalCSVData []HospitalCSV
	b, _ := ioutil.ReadFile("./data/nagoya/hospital.csv")
	// ユーザー定義型スライスにマッピング
	csvutil.Unmarshal(b, &hospitalCSVData)
	
	for _, v := range hospitalCSVData{
		hospitalData = append(hospitalData, Hospital{
			HospitalID : v.HospitalID,
			Name: v.Name,
			Code : v.Code,
			Latitude : convertToCoord(v.Latitude),
			Longitude : convertToCoord(v.Longitude),
		})
	}

	hospitalMap := make(map[string]Hospital)
	for _, v := range hospitalData{
		hospitalMap[v.HospitalID] = v
	}
	return hospitalMap
}

func getRandomNodeLinkID(node Node)string{
	linkIDSlice := []string{node.LinkID1}
	if node.LinkID2 != "" { linkIDSlice = append(linkIDSlice, node.LinkID2) }
	if node.LinkID3 != "" { linkIDSlice = append(linkIDSlice, node.LinkID3) }
	if node.LinkID4 != "" { linkIDSlice = append(linkIDSlice, node.LinkID4) }
	if node.LinkID5 != "" { linkIDSlice = append(linkIDSlice, node.LinkID5) }
	if node.LinkID6 != "" { linkIDSlice = append(linkIDSlice, node.LinkID6) }
	linkID := linkIDSlice[rand.Intn(len(linkIDSlice))]
	return linkID
}

func createRoute(transitNum int, nodeData map[string]Node, linkData map[string]Link) (*api.Coord, *api.Coord, []*api.Coord){
	// departure
	keys := make([]string, len(nodeData))
	i := 0
	for k := range nodeData {
		keys[i] = k
		i++
	}
	randkey := keys[rand.Intn(len(nodeData))]
	departureNode := nodeData[randkey]
	departure := &api.Coord{Latitude: departureNode.Latitude, Longitude: departureNode.Longitude}
	
	// transitpoints
	transitPoints := []*api.Coord{}
	tgtNode := departureNode
	for i := 0; i < transitNum; i++ {
		//log.Print(tgtNode.NodeID)
		nextLink := linkData[getRandomNodeLinkID(tgtNode)]
		nextNode := nodeData[nextLink.StartNodeID]
		if tgtNode.Latitude == nextNode.Latitude && tgtNode.Longitude == nextNode.Longitude{
			nextNode = nodeData[nextLink.EndNodeID]  // 同じNodeに戻らないようにする
		}
		transitPoints = append(transitPoints, &api.Coord{
			Latitude: nextNode.Latitude,
			Longitude: nextNode.Longitude,
		})
		tgtNode = nextNode
	}
	// destination
	destinationLink := linkData[tgtNode.LinkID1]
	destinationNode := nodeData[destinationLink.EndNodeID]
	destination := &api.Coord{Latitude: destinationNode.Latitude, Longitude: destinationNode.Longitude}
	return departure, destination, transitPoints
}