package algorithm

import (
	"math"
	"math/rand"

	"encoding/json"

	api "github.com/RuiHirano/flow_beta/api"
	rvo "github.com/RuiHirano/rvo2-go/src/rvosimulator"
)

var (
	//sim       *rvo.RVOSimulator
	//logger    *util.Logger
	//routeName string

//fcs *geojson.FeatureCollection
)


type ModelParam struct{
	Radius float64  `json:"radius"` // 半径何m?以内にいる人が接触と判断するか
	Rate float64 `json:"rate"`// 何%で感染するか
	DefaultInfRate float64 `json:"defaultInfRate"` // 最初のエージェントの何％が感染から始まるか
}

type AgentParam struct{
	Status string  `json:"status"`  // S: まだ感染していない人, I: 感染しており、他者に感染させる能力を持つ人, R: 病気から回復して免疫を得た人、あるいは死亡した人
	Move int  `json:"move"` // 0: ランダムに動く   1: 止まる
}

type Infection struct {
	Agents []*api.Agent
	ModelParam *ModelParam
	Area       *api.Area
}

func NewInfection(param *ModelParam) *Infection {
	r := &Infection{
		ModelParam: param,
	}
	return r
}

func (inf *Infection) SimulationRequest(simReq *api.SimulatorRequest) {
	simType := simReq.GetType()
	if simType == "SET_PARAM"{
		data := simReq.GetData()
		var param *ModelParam
		json.Unmarshal([]byte(data), &param)
		inf.ModelParam = param
		logger.Success("Set Param: radius %v, rate %v, defaultInfRate %v", param.Radius, param.Rate, param.DefaultInfRate, data)
	}
}

func (inf *Infection) SetAgents(agents []*api.Agent) {
	inf.Agents = agents
}

func (inf *Infection) SetArea(area *api.Area) {
	inf.Area = area
}

// CalcDirectionAndDistance: 目的地までの距離と角度を計算する関数
func (inf *Infection) CalcDirectionAndDistance(startCoord *api.Coord, goalCoord *api.Coord) (float64, float64) {

	r := 6378137 // equatorial radius
	sLat := startCoord.Latitude * math.Pi / 180
	sLon := startCoord.Longitude * math.Pi / 180
	gLat := goalCoord.Latitude * math.Pi / 180
	gLon := goalCoord.Longitude * math.Pi / 180
	dLon := gLon - sLon
	dLat := gLat - sLat
	cLat := (sLat + gLat) / 2
	dx := float64(r) * float64(dLon) * math.Cos(float64(cLat))
	dy := float64(r) * float64(dLat)

	distance := math.Sqrt(math.Pow(dx, 2) + math.Pow(dy, 2))
	direction := float64(0)
	if dx != 0 && dy != 0 {
		direction = math.Atan2(dy, dx) * 180 / math.Pi
	}

	return direction, distance
}

// DecideNextTransit: 次の経由地を求める関数
func (inf *Infection) DecideNextTransit(nextTransit *api.Coord, transitPoint []*api.Coord, distance float64, destination *api.Coord) *api.Coord {

	// 距離が5m以下の場合
	if distance < 10 {
		if nextTransit != destination {
			for i, tPoint := range transitPoint {
				if tPoint.Longitude == nextTransit.Longitude && tPoint.Latitude == nextTransit.Latitude {
					if i+1 == len(transitPoint) {
						// すべての経由地を通った場合、nextTransitをdestinationにする
						nextTransit = destination
						logger.Info("Destination %v %v", nextTransit, len(transitPoint))
					} else {
						// 次の経由地を設定する
						nextTransit = transitPoint[i+1]
						logger.Info("Next Transit %v %v", nextTransit, len(transitPoint))
					}
				}
			}
		} else {
			logger.Info("Arrived")
		}
	}
	return nextTransit
}

// DecideNextTransit: 次の経由地を求める関数
func (inf *Infection) DecideNextTransit2(position *api.Coord, nextTransit *api.Coord, transitPoint []*api.Coord) *api.Coord {

	// 距離を確認
	_, distance := inf.CalcDirectionAndDistance(position, nextTransit)

	// 距離が5m以下の場合
	if distance < 5 {
		
		if nextTransit != destination {
			for i, tPoint := range transitPoint {
				if tPoint.Longitude == nextTransit.Longitude && tPoint.Latitude == nextTransit.Latitude {
					if i+1 == len(transitPoint) {
						// すべての経由地を通った場合、nextTransitをdestinationにする
						nextTransit = destination
						logger.Info("Destination %v %v", nextTransit, len(transitPoint))
					} else {
						// 次の経由地を設定する
						nextTransit = transitPoint[i+1]
						logger.Info("Next Transit %v %v", nextTransit, len(transitPoint))
					}
				}
			}
		} else {
			logger.Info("Arrived")
		}
	}
	return nextTransit
}

// GetNextTransit: 次の経由地を求める関数
func (inf *Infection) GetNextTransit(nextTransit *api.Coord, distance float64) *api.Coord {
	newNextTransit := nextTransit
	//logger.Error("Name: %v, Distance %v\n", routeName, distance)
	// 距離が5m以下の場合
	/*if distance < 10 {
		routes := GetRoutes2()
		for _, route := range routes {
			if route.Point.Longitude == nextTransit.Longitude && route.Point.Latitude == nextTransit.Latitude {
				index := rand.Intn(len(route.NeighborPoints))
				nextRoute := route.NeighborPoints[index]
				newNextTransit = nextRoute.Point
				routeName = nextRoute.Name
				//logger.Warn("Name: %v, Index %v\n", routeName, index)
				break
			}
		}
	}*/
	return newNextTransit
}

// SetupScenario: Scenarioを設定する関数
func (inf *Infection) SetupScenario() {
	// Set Agent
	for _, agentInfo := range inf.Agents {

		position := &rvo.Vector2{X: agentInfo.Route.Position.Longitude, Y: agentInfo.Route.Position.Latitude}
		goal := &rvo.Vector2{X: agentInfo.Route.NextTransit.Longitude, Y: agentInfo.Route.NextTransit.Latitude}

		// Agentを追加
		id, _ := sim.AddDefaultAgent(position)

		// 目的地を設定
		sim.SetAgentGoal(id, goal)

		// エージェントの速度方向ベクトルを設定
		goalVector := sim.GetAgentGoalVector(id)
		sim.SetAgentPrefVelocity(id, goalVector)
		//sim.SetAgentMaxSpeed(id, float64(api.MaxSpeed))
	}
}

func (inf *Infection) CalcNextAgents() []*api.Agent {

	currentAgents := inf.Agents

	timeStep := 0.1
	neighborDist := 0.00003 // どのくらいの距離の相手をNeighborと認識するか?Neighborとの距離をどのくらいに保つか？ぶつかったと認識する距離？
	maxneighbors := 3       // 周り何体を計算対象とするか
	timeHorizon := 1.0
	timeHorizonObst := 1.0
	radius := 0.00001  // エージェントの半径
	maxSpeed := 0.0004 // エージェントの最大スピード
	sim = rvo.NewRVOSimulator(timeStep, neighborDist, maxneighbors, timeHorizon, timeHorizonObst, radius, maxSpeed, &rvo.Vector2{X: 0, Y: 0})

	// scenario設定
	inf.SetupScenario()

	// Stepを進める
	sim.DoStep()

	// 管理エリアのエージェントのみを抽出
	nextControlAgents := make([]*api.Agent, 0)
	for rvoId, agentInfo := range currentAgents {
		// 管理エリア内のエージェントのみ抽出
		position := agentInfo.Route.Position
		if IsAgentInArea(position, inf.Area.ControlArea) {
			destination := agentInfo.Route.Destination

			// rvoの位置情報を緯度経度に変換する
			rvoAgent := sim.GetAgent(int(rvoId))
			rvoAgentPosition := rvoAgent.Position

			// infection探索
			var data *AgentParam
			json.Unmarshal([]byte(agentInfo.Data), &data)
			if data.Status == "S"{
				// 自身が感染しておらず、半径rの周りに感染している人がいれば、自身も感染する
				rvoNeighbors := rvoAgent.AgentNeighbors
				for _, rvoNeighbor := range rvoNeighbors{
					neighbor := currentAgents[rvoNeighbor.Agent.ID]
					var neighborData *AgentParam
					json.Unmarshal([]byte(neighbor.Data), &neighborData)
					// 半径rに感染している人がいる場合
					if neighborData.Status == "I" && math.Sqrt(rvoNeighbor.DistSq) < inf.ModelParam.Radius{
						// xの確率で感染する
						if rand.Float64() < inf.ModelParam.Rate {
							// 感染
							data.Status = "I"
						}
					}
				}
			}

			position := &api.Coord{
				Latitude:  rvoAgentPosition.Y,
				Longitude: rvoAgentPosition.X,
			}

			// 現在の位置とゴールとの距離と角度を求める (度, m))
			//_, distance := inf.CalcDirectionAndDistance(position, agentInfo.Route.NextTransit)
			// 次の経由地nextTransitを求める
			//nextTransit := inf.DecideNextTransit(agentInfo.Route.NextTransit, agentInfo.Route.TransitPoints, distance, destination)
			nextTransit := inf.DecideNextTransit2(position, agentInfo.Route.NextTransit ,agentInfo.Route.TransitPoints)
			//nextTransit := agentInfo.Route.NextTransit
			//nextTransit := inf.GetNextTransit(agentInfo.Route.NextTransit, distance)

			goalVector := sim.GetAgentGoalVector(int(rvoId))
			direction := math.Atan2(goalVector.Y, goalVector.X)
			speed := agentInfo.Route.Speed

			nextRoute := &api.Route{
				Position:      position,
				Direction:     direction,
				Speed:         speed,
				Destination:   destination,
				Departure:     agentInfo.Route.Departure,
				TransitPoints: agentInfo.Route.TransitPoints,
				NextTransit:   nextTransit,
				TotalDistance: agentInfo.Route.TotalDistance,
				RequiredTime:  agentInfo.Route.RequiredTime,
			}

			dataStr, _ := json.Marshal(data)

			nextControlAgent := &api.Agent{
				Id:    agentInfo.Id,
				Type:  agentInfo.Type,
				Route: nextRoute,
				Data: string(dataStr),
			}

			nextControlAgents = append(nextControlAgents, nextControlAgent)
		}
	}
	logger.Info("DoSteped! %v", len(nextControlAgents))

	return nextControlAgents
}

// エージェントがエリアの中にいるかどうか
/*func IsAgentInArea(position *api.Coord, areaCoords []*api.Coord) bool {
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
}*/