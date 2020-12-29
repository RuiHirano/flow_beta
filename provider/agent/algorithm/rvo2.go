package algorithm

import (
	"math"

	//"github.com/paulmach/orb"
	//"github.com/paulmach/orb/geojson"

	//monitor "github.com/RuiHirano/rvo2-go/monitor"

	//"io/ioutil"
	//"log"
	"math/rand"

	api "github.com/RuiHirano/flow_beta/api"
	util "github.com/RuiHirano/flow_beta/util"
	rvo "github.com/RuiHirano/rvo2-go/src/rvosimulator"
)

var (
	sim       *rvo.RVOSimulator
	logger    *util.Logger
	routeName string

//fcs *geojson.FeatureCollection
)

func init() {
	logger = util.NewLogger()
	routeName = ""
}

type RVO2Route struct {
	Agents []*api.Agent
	Area   *api.Area
}

func NewRVO2Route(agents []*api.Agent, area *api.Area) *RVO2Route {

	r := &RVO2Route{
		Agents: agents,
		Area:   area,
	}
	return r
}

// CalcDirectionAndDistance: 目的地までの距離と角度を計算する関数
func (rvo2route *RVO2Route) CalcDirectionAndDistance(startCoord *api.Coord, goalCoord *api.Coord) (float64, float64) {

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
func (rvo2route *RVO2Route) DecideNextTransit(nextTransit *api.Coord, transitPoint []*api.Coord, distance float64, destination *api.Coord) *api.Coord {

	// 距離が5m以下の場合
	if distance < 150 {
		if nextTransit != destination {
			for i, tPoint := range transitPoint {
				if tPoint.Longitude == nextTransit.Longitude && tPoint.Latitude == nextTransit.Latitude {
					if i+1 == len(transitPoint) {
						// すべての経由地を通った場合、nextTransitをdestinationにする
						nextTransit = destination
					} else {
						// 次の経由地を設定する
						nextTransit = transitPoint[i+1]
					}
				}
			}
		} else {
			//fmt.Printf("arrived!")
		}
	}
	return nextTransit
}

// GetNextTransit: 次の経由地を求める関数
func (rvo2route *RVO2Route) GetNextTransit(nextTransit *api.Coord, distance float64) *api.Coord {
	newNextTransit := nextTransit
	//logger.Error("Name: %v, Distance %v\n", routeName, distance)
	// 距離が5m以下の場合
	if distance < 10 {
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
	}
	return newNextTransit
}

// SetupScenario: Scenarioを設定する関数
func (rvo2route *RVO2Route) SetupScenario() {
	// Set Agent
	for _, agentInfo := range rvo2route.Agents {

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

func (rvo2route *RVO2Route) CalcNextAgents() []*api.Agent {

	currentAgents := rvo2route.Agents

	timeStep := 0.1
	neighborDist := 0.00003 // どのくらいの距離の相手をNeighborと認識するか?Neighborとの距離をどのくらいに保つか？ぶつかったと認識する距離？
	maxneighbors := 3       // 周り何体を計算対象とするか
	timeHorizon := 1.0
	timeHorizonObst := 1.0
	radius := 0.00001  // エージェントの半径
	maxSpeed := 0.0004 // エージェントの最大スピード
	sim = rvo.NewRVOSimulator(timeStep, neighborDist, maxneighbors, timeHorizon, timeHorizonObst, radius, maxSpeed, &rvo.Vector2{X: 0, Y: 0})

	// scenario設定
	rvo2route.SetupScenario()

	// Stepを進める
	sim.DoStep()

	// 管理エリアのエージェントのみを抽出
	nextControlAgents := make([]*api.Agent, 0)
	for rvoId, agentInfo := range currentAgents {
		// 管理エリア内のエージェントのみ抽出
		position := agentInfo.Route.Position
		if IsAgentInArea(position, rvo2route.Area.ControlArea) {
			destination := agentInfo.Route.Destination

			// rvoの位置情報を緯度経度に変換する
			rvoAgentPosition := sim.GetAgentPosition(int(rvoId))

			nextCoord := &api.Coord{
				Latitude:  rvoAgentPosition.Y,
				Longitude: rvoAgentPosition.X,
			}

			// 現在の位置とゴールとの距離と角度を求める (度, m))
			_, distance := rvo2route.CalcDirectionAndDistance(nextCoord, agentInfo.Route.NextTransit)
			// 次の経由地nextTransitを求める
			//nextTransit := rvo2route.DecideNextTransit(agentInfo.Route.NextTransit, agentInfo.Route.TransitPoints, distance, destination)
			//nextTransit := agentInfo.Route.NextTransit
			nextTransit := rvo2route.GetNextTransit(agentInfo.Route.NextTransit, distance)

			goalVector := sim.GetAgentGoalVector(int(rvoId))
			direction := math.Atan2(goalVector.Y, goalVector.X)
			speed := agentInfo.Route.Speed

			nextRoute := &api.Route{
				Position:      nextCoord,
				Direction:     direction,
				Speed:         speed,
				Destination:   destination,
				Departure:     agentInfo.Route.Departure,
				TransitPoints: agentInfo.Route.TransitPoints,
				NextTransit:   nextTransit,
				TotalDistance: agentInfo.Route.TotalDistance,
				RequiredTime:  agentInfo.Route.RequiredTime,
			}

			nextControlAgent := &api.Agent{
				Id:    agentInfo.Id,
				Type:  agentInfo.Type,
				Route: nextRoute,
				Data: agentInfo.Data,
			}

			nextControlAgents = append(nextControlAgents, nextControlAgent)
		}
	}
	logger.Info("DoSteped! %v", len(nextControlAgents))

	return nextControlAgents
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

type RoutePoint struct {
	Id             uint64
	Name           string
	Point          *api.Coord
	NeighborPoints []*RoutePoint
}

func GetRouteMap() map[uint64][]*RoutePoint {
	routeMap := make(map[uint64][]*RoutePoint)
	routeMap[0] = []*RoutePoint{{Id: 1, Name: "enterance", Point: &api.Coord{Longitude: 136.974688, Latitude: 35.158228}}}
	routeMap[1] = []*RoutePoint{
		{Id: 0, Name: "gate", Point: &api.Coord{Longitude: 136.974024, Latitude: 35.158995}},
		{Id: 2, Name: "rightEnt", Point: &api.Coord{Longitude: 136.974645, Latitude: 35.157958}},
		{Id: 3, Name: "leftEnt", Point: &api.Coord{Longitude: 136.974938, Latitude: 35.158164}},
	}
	routeMap[2] = []*RoutePoint{
		{Id: 1, Name: "enterance", Point: &api.Coord{Longitude: 136.974688, Latitude: 35.158228}},
		{Id: 4, Name: "road1", Point: &api.Coord{Longitude: 136.974864, Latitude: 35.157823}},
	}
	routeMap[3] = []*RoutePoint{
		{Id: 1, Name: "enterance", Point: &api.Coord{Longitude: 136.974688, Latitude: 35.158228}},
		{Id: 5, Name: "road2", Point: &api.Coord{Longitude: 136.975054, Latitude: 35.158001}},
	}
	routeMap[4] = []*RoutePoint{
		{Id: 2, Name: "rightEnt", Point: &api.Coord{Longitude: 136.974645, Latitude: 35.157958}},
		{Id: 5, Name: "road2", Point: &api.Coord{Longitude: 136.975054, Latitude: 35.158001}},
		{Id: 6, Name: "road3", Point: &api.Coord{Longitude: 136.975517, Latitude: 35.157096}},
	}
	routeMap[5] = []*RoutePoint{
		{Id: 3, Name: "leftEnt", Point: &api.Coord{Longitude: 136.974938, Latitude: 35.158164}},
		{Id: 4, Name: "road1", Point: &api.Coord{Longitude: 136.974864, Latitude: 35.157823}},
	}
	routeMap[6] = []*RoutePoint{
		{Id: 7, Name: "road4", Point: &api.Coord{Longitude: 136.975872, Latitude: 35.156678}},
		{Id: 4, Name: "road1", Point: &api.Coord{Longitude: 136.974864, Latitude: 35.157823}},
	}

	return routeMap
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
		//
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
		//
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

func GetRoutes2() []*RoutePoint {
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
				{Id: 2, Name: "rightEnt", Point: &api.Coord{Longitude: 136.974645, Latitude: 35.157958}},
				{Id: 3, Name: "leftEnt", Point: &api.Coord{Longitude: 136.974938, Latitude: 35.158164}},
			},
		},
		{
			Id: 2, Name: "rightEnt", Point: &api.Coord{Longitude: 136.974645, Latitude: 35.157958},
			NeighborPoints: []*RoutePoint{
				{Id: 4, Name: "road1", Point: &api.Coord{Longitude: 136.974864, Latitude: 35.157823}},
			},
		},
		{
			Id: 3, Name: "leftEnt", Point: &api.Coord{Longitude: 136.974938, Latitude: 35.158164},
			NeighborPoints: []*RoutePoint{
				{Id: 5, Name: "road2", Point: &api.Coord{Longitude: 136.975054, Latitude: 35.158001}},
				{Id: 17, Name: "north1", Point: &api.Coord{Longitude: 136.976395, Latitude: 35.158410}},
			},
		},
		{
			Id: 4, Name: "road1", Point: &api.Coord{Longitude: 136.974864, Latitude: 35.157823},
			NeighborPoints: []*RoutePoint{
				{Id: 5, Name: "road2", Point: &api.Coord{Longitude: 136.975054, Latitude: 35.158001}},
				{Id: 6, Name: "road3", Point: &api.Coord{Longitude: 136.975517, Latitude: 35.157096}},
			},
		},
		{
			Id: 5, Name: "road2", Point: &api.Coord{Longitude: 136.975054, Latitude: 35.158001},
			NeighborPoints: []*RoutePoint{
				{Id: 4, Name: "road1", Point: &api.Coord{Longitude: 136.974864, Latitude: 35.157823}},
				{Id: 6, Name: "road3", Point: &api.Coord{Longitude: 136.975517, Latitude: 35.157096}},
			},
		},
		{
			Id: 6, Name: "road3", Point: &api.Coord{Longitude: 136.975517, Latitude: 35.157096},
			NeighborPoints: []*RoutePoint{
				{Id: 7, Name: "road4", Point: &api.Coord{Longitude: 136.975872, Latitude: 35.156678}},
			},
		},
		{
			Id: 7, Name: "road4", Point: &api.Coord{Longitude: 136.975872, Latitude: 35.156678},
			NeighborPoints: []*RoutePoint{
				{Id: 8, Name: "road5", Point: &api.Coord{Longitude: 136.976314, Latitude: 35.156757}},
				{Id: 10, Name: "burger", Point: &api.Coord{Longitude: 136.976960, Latitude: 35.155697}},
			},
		},
		{
			Id: 8, Name: "road5", Point: &api.Coord{Longitude: 136.976314, Latitude: 35.156757},
			NeighborPoints: []*RoutePoint{
				{Id: 9, Name: "toilet", Point: &api.Coord{Longitude: 136.977261, Latitude: 35.155951}},
			},
		},
		{
			Id: 9, Name: "toilet", Point: &api.Coord{Longitude: 136.977261, Latitude: 35.155951},
			NeighborPoints: []*RoutePoint{
				{Id: 10, Name: "burger", Point: &api.Coord{Longitude: 136.976960, Latitude: 35.155697}},
			},
		},
		{
			Id: 10, Name: "burger", Point: &api.Coord{Longitude: 136.976960, Latitude: 35.155697},
			NeighborPoints: []*RoutePoint{
				{Id: 11, Name: "lake1", Point: &api.Coord{Longitude: 136.978217, Latitude: 35.155266}},
			},
		},
		{
			Id: 11, Name: "lake1", Point: &api.Coord{Longitude: 136.978217, Latitude: 35.155266},
			NeighborPoints: []*RoutePoint{
				{Id: 12, Name: "lake2", Point: &api.Coord{Longitude: 136.978623, Latitude: 35.155855}},
				{Id: 16, Name: "lake6", Point: &api.Coord{Longitude: 136.978297, Latitude: 35.154755}},
			},
		},
		{
			Id: 12, Name: "lake2", Point: &api.Coord{Longitude: 136.978623, Latitude: 35.155855},
			NeighborPoints: []*RoutePoint{
				{Id: 13, Name: "lake3", Point: &api.Coord{Longitude: 136.979657, Latitude: 35.155659}},
				{Id: 11, Name: "lake1", Point: &api.Coord{Longitude: 136.978217, Latitude: 35.155266}},
			},
		},
		{
			Id: 13, Name: "lake3", Point: &api.Coord{Longitude: 136.979657, Latitude: 35.155659},
			NeighborPoints: []*RoutePoint{
				{Id: 14, Name: "lake4", Point: &api.Coord{Longitude: 136.980489, Latitude: 35.154484}},
				{Id: 12, Name: "lake2", Point: &api.Coord{Longitude: 136.978623, Latitude: 35.155855}},
				{Id: 26, Name: "east6", Point: &api.Coord{Longitude: 136.984100, Latitude: 35.153693}},
				{Id: 22, Name: "east1", Point: &api.Coord{Longitude: 136.981124, Latitude: 35.157283}},
				{Id: 27, Name: "east-in1", Point: &api.Coord{Longitude: 136.982804, Latitude: 35.154175}},
			},
		},
		{
			Id: 14, Name: "lake4", Point: &api.Coord{Longitude: 136.980489, Latitude: 35.154484},
			NeighborPoints: []*RoutePoint{
				{Id: 15, Name: "lake5", Point: &api.Coord{Longitude: 136.980143, Latitude: 35.153869}},
				{Id: 13, Name: "lake3", Point: &api.Coord{Longitude: 136.979657, Latitude: 35.155659}},
			},
		},
		{
			Id: 15, Name: "lake5", Point: &api.Coord{Longitude: 136.980143, Latitude: 35.153869},
			NeighborPoints: []*RoutePoint{
				{Id: 16, Name: "lake6", Point: &api.Coord{Longitude: 136.978297, Latitude: 35.154755}},
				{Id: 15, Name: "lake5", Point: &api.Coord{Longitude: 136.980143, Latitude: 35.153869}},
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
				{Id: 26, Name: "east6", Point: &api.Coord{Longitude: 136.984100, Latitude: 35.153693}},
			},
		},
		//
		{
			Id: 23, Name: "east2", Point: &api.Coord{Longitude: 136.984350, Latitude: 35.157271},
			NeighborPoints: []*RoutePoint{
				{Id: 24, Name: "east3", Point: &api.Coord{Longitude: 136.987567, Latitude: 35.158233}},
				{Id: 22, Name: "east1", Point: &api.Coord{Longitude: 136.981124, Latitude: 35.157283}},
			},
		},
		{
			Id: 24, Name: "east3", Point: &api.Coord{Longitude: 136.987567, Latitude: 35.158233},
			NeighborPoints: []*RoutePoint{
				{Id: 25, Name: "east4", Point: &api.Coord{Longitude: 136.988522, Latitude: 35.157286}},
				{Id: 23, Name: "east2", Point: &api.Coord{Longitude: 136.984350, Latitude: 35.157271}},
			},
		},
		{
			Id: 25, Name: "east4", Point: &api.Coord{Longitude: 136.988522, Latitude: 35.157286},
			NeighborPoints: []*RoutePoint{
				{Id: 25, Name: "east5", Point: &api.Coord{Longitude: 136.988355, Latitude: 35.155838}},
				{Id: 24, Name: "east3", Point: &api.Coord{Longitude: 136.987567, Latitude: 35.158233}},
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
				{Id: 13, Name: "lake3", Point: &api.Coord{Longitude: 136.979657, Latitude: 35.155659}},
				{Id: 27, Name: "east-in1", Point: &api.Coord{Longitude: 136.982804, Latitude: 35.154175}},
				{Id: 22, Name: "east1", Point: &api.Coord{Longitude: 136.981124, Latitude: 35.157283}},
			},
		},
		//
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

// {Id: 0, Name: "gate", Point: &api.Coord{Longitude: 136.974024, Latitude: 35.158995}
// {Id: 1, Name: "enterance", Point: &api.Coord{Longitude: 136.974688, Latitude: 35.158228}},
// {Id: 2, Name: "rightEnt", Point: &api.Coord{Longitude: 136.974645, Latitude: 35.157958}},
// {Id: 3, Name: "leftEnt", Point: &api.Coord{Longitude: 136.974938, Latitude: 35.158164}},
// {Id: 4, Name: "road1", Point: &api.Coord{Longitude: 136.974864, Latitude: 35.157823}},
// {Id: 5, Name: "road2", Point: &api.Coord{Longitude: 136.975054, Latitude: 35.158001}},
// {Id: 6, Name: "road3", Point: &api.Coord{Longitude: 136.975517, Latitude: 35.157096}},
// {Id: 7, Name: "road4", Point: &api.Coord{Longitude: 136.975872, Latitude: 35.156678}},
