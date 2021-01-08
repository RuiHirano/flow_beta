package main

import (
	"encoding/json"
	"math"

	"math/rand"

	api "github.com/RuiHirano/flow_beta/api"
	algo "github.com/RuiHirano/flow_beta/provider/agent/algorithm"
	"github.com/google/uuid"
)

var (
	infection *algo.Infection
)
func init(){	

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
	maxLat, maxLon, minLat, minLon := GetCoordRange(myArea.ControlArea)
	for range agents{

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
				Position:      position,
				Direction:     30,
				Speed:         60,
				Departure:     position,
				Destination:   destination,
				TransitPoints: transitPoints,
				NextTransit:   transitPoint,
			},
			Data: string(agentModelParamJson),
		})
	}
	return expAgents
}