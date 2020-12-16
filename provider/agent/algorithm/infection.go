package algorithm

import (
	"fmt"
	"math"

	api "github.com/RuiHirano/flow_beta/api"
)

type ModelParam struct{
	Radius int   // 半径何m以内にいる人が接触と判断するか
	Rate float64 // 何%が感染するか
}

type AgentParam struct{
	Status string  // S: まだ感染していない人, I: 感染しており、他者に感染させる能力を持つ人, R: 病気から回復して免疫を得た人、あるいは死亡した人
	Move int // 0: ランダムに動く   1: 止まる
}

type Infection struct {
	Agents []*api.Agent
	ModelParam *ModelParam
}

func NewInfection(param *ModelParam) *Infection {
	r := &Infection{
		Agents: []*api.Agent{},
		ModelParam: param,
	}
	return r
}

func (inf *Infection) AddAgents(agents []*api.Agent){
	inf.Agents = agents
}

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

// TODO: Why Calc Error ? newLat=nan and newLon = inf
func (inf *Infection) CalcMovedPosition(currentPosition *api.Coord, goalPosition *api.Coord, distance float64, speed float64) *api.Coord {

	sLat := currentPosition.Latitude
	sLon := currentPosition.Longitude
	gLat := goalPosition.Latitude
	gLon := goalPosition.Longitude
	// 割合
	x := speed * 1000 / 3600 / distance

	newLat := sLat + (gLat-sLat)*x
	newLon := sLon + (gLon-sLon)*x

	nextPosition := &api.Coord{
		Latitude:  newLat,
		Longitude: newLon,
	}

	return nextPosition
}

// DecideNextTransit: 次の経由地を決める関数
func (inf *Infection) DecideNextTransit(nextTransit *api.Coord, transitPoint []*api.Coord, distance float64, destination *api.Coord) *api.Coord {
	// 距離が5m以下の場合
	if distance < 5 {
		if nextTransit != destination {
			for i, tPoint := range transitPoint {
				if tPoint.Longitude == nextTransit.Longitude && tPoint.Latitude == nextTransit.Latitude {
					if i+1 == len(transitPoint) {
						// すべての経由地を通った場合、nilにする
						nextTransit = destination
					} else {
						// 次の経由地を設定する
						nextTransit = transitPoint[i+1]
					}
				}
			}
		} else {
			fmt.Printf("\x1b[30m\x1b[47m Arrived Destination! \x1b[0m\n")
		}
	}
	return nextTransit
}

// CalcNextRoute：次の時刻のRouteを計算する関数
func (inf *Infection) CalcNextRoute(agentInfo *api.Agent) *api.Route {

	route := agentInfo.Route
	speed := route.Speed
	currentPosition := route.Position
	nextTransit := route.NextTransit
	transitPoints := route.TransitPoints
	destination := route.Destination
	// passed all transit point
	//if nextTransit != nil {
	//	destination = nextTransit
	//}

	// 現在位置と目標位置との距離と角度を計算
	direction, distance := inf.CalcDirectionAndDistance(currentPosition, nextTransit)

	// 次の時刻のPositionを計算
	nextPosition := inf.CalcMovedPosition(currentPosition, nextTransit, distance, speed)

	// 経由地に到着していれば、目標位置を次の経由地に更新する
	nextTransit = inf.DecideNextTransit(nextTransit, transitPoints, distance, destination)

	//fmt.Printf("\x1b[30m\x1b[47m Position %v, NextTransit: %v, NextTransit: %v, Direction: %v, Distance: %v \x1b[0m\n", currentPosition, nextTransit, destination, direction, distance)
	//fmt.Printf("\x1b[30m\x1b[47m 上下:  %v, 左右: %v \x1b[0m\n", nextTransit.Lat-currentPosition.Lat, nextTransit.Lon-currentPosition.Lon)

	//} else {
	//	log.Printf("\x1b[30m\x1b[47m LOCATION CULC ERROR %v \x1b[0m\n", nextPosition)
	//}

	nextRoute := &api.Route{
		Position:      nextPosition,
		Direction:     direction,
		Speed:         speed,
		Destination:   route.Destination,
		Departure:     route.Departure,
		TransitPoints: transitPoints,
		NextTransit:   nextTransit,
		TotalDistance: route.TotalDistance,
		RequiredTime:  route.RequiredTime,
	}

	return nextRoute
}



// CalcNextAgents: 次の時刻のエージェントを取得する関数
func (inf *Infection) CalcAgents(agents []*api.Agent) []*api.Agent {

	nextControlAgents := make([]*api.Agent, 0)

	for _, agentInfo := range agents {
		// 次の時刻のRouteを計算
		nextRoute := inf.CalcNextRoute(agentInfo)


		nextControlAgent := &api.Agent{
			Id:    agentInfo.Id,
			Type:  agentInfo.Type,
			Route: nextRoute,
		}
		// Agent追加
		nextControlAgents = append(nextControlAgents, nextControlAgent)
		// 自エリアにいる場合、次のルートを計算する
		/*if IsAgentInArea(agentInfo.Route.Position, inf.Area.ControlArea) {

			// 次の時刻のRouteを計算
			nextRoute := inf.CalcNextRoute(agentInfo, inf.SameAreaAgents)

			nextControlAgent := &api.Agent{
				Id:    agentInfo.Id,
				Type:  agentInfo.Type,
				Route: nextRoute,
			}
			// Agent追加
			nextControlAgents = append(nextControlAgents, nextControlAgent)
		}*/
	}
	return nextControlAgents
}

// エージェントがエリアの中にいるかどうか
/*func IsAgentInArea(position *api.Coord, areaCoords []*api.Coord) bool {
	lat := position.Latitude
	lon := position.Longitude
	maxLat, maxLon, minLat, minLon := simutil.GetCoordRange(areaCoords)
	if minLat < lat && lat < maxLat && minLon < lon && lon < maxLon {
		return true
	} else {
		return false
	}
}*/
