package algorithm

import (
	"math"

	api "github.com/RuiHirano/flow_beta/api"
)

type SimpleRoute struct {
	Agents []*api.Agent
}

func NewSimpleRoute(agents []*api.Agent) *SimpleRoute {
	r := &SimpleRoute{
		Agents: agents,
	}
	return r
}

func (simple *SimpleRoute) CalcDirectionAndDistance(startCoord *api.Coord, goalCoord *api.Coord) (float64, float64) {

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
func (simple *SimpleRoute) CalcMovedPosition(currentPosition *api.Coord, goalPosition *api.Coord, distance float64, speed float64) *api.Coord {

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
/*func (simple *SimpleRoute) DecideNextTransit(nextTransit *api.Coord, transitPoint []*api.Coord, distance float64, destination *api.Coord) *api.Coord {
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
}*/

// DecideNextTransit: 次の経由地を求める関数
func (simple *SimpleRoute) DecideNextTransit(position *api.Coord, nextTransit *api.TransitPoint, transitPoints []*api.TransitPoint) *api.TransitPoint {

	// 距離を確認
	_, distance := simple.CalcDirectionAndDistance(position, nextTransit.Coord)

	// 距離が5m以下の場合
	if distance < 5 {
		nowNodeId := nextTransit.Id
		for i, v := range transitPoints{
			if nowNodeId == v.Id{
				if i == len(transitPoints){
					logger.Info("Arrived")
				}else{
					nextTransit = transitPoints[i+1]
				}
			}
		}
	}
	return nextTransit
}

// CalcNextRoute：次の時刻のRouteを計算する関数
func (simple *SimpleRoute) CalcNextRoute(agentInfo *api.Agent) *api.Route {

	route := agentInfo.Route
	speed := route.Speed
	currentPosition := route.Position
	nextTransit := route.NextTransit
	transitPoints := route.TransitPoints
	//destination := route.Destination
	// passed all transit point
	//if nextTransit != nil {
	//	destination = nextTransit
	//}

	// 現在位置と目標位置との距離と角度を計算
	direction, distance := simple.CalcDirectionAndDistance(currentPosition, nextTransit.Coord)

	// 次の時刻のPositionを計算
	position := simple.CalcMovedPosition(currentPosition, nextTransit.Coord, distance, speed)

	// 経由地に到着していれば、目標位置を次の経由地に更新する
	nextTransit = simple.DecideNextTransit(position, agentInfo.Route.NextTransit ,agentInfo.Route.TransitPoints)

	//fmt.Printf("\x1b[30m\x1b[47m Position %v, NextTransit: %v, NextTransit: %v, Direction: %v, Distance: %v \x1b[0m\n", currentPosition, nextTransit, destination, direction, distance)
	//fmt.Printf("\x1b[30m\x1b[47m 上下:  %v, 左右: %v \x1b[0m\n", nextTransit.Lat-currentPosition.Lat, nextTransit.Lon-currentPosition.Lon)

	//} else {
	//	log.Printf("\x1b[30m\x1b[47m LOCATION CULC ERROR %v \x1b[0m\n", nextPosition)
	//}

	nextRoute := &api.Route{
		Position:      position,
		Direction:     direction,
		Speed:         speed,
		TransitPoints: transitPoints,
		NextTransit:   nextTransit,
	}

	return nextRoute
}

// CalcNextAgents: 次の時刻のエージェントを取得する関数
func (simple *SimpleRoute) CalcNextAgents() []*api.Agent {

	nextControlAgents := make([]*api.Agent, 0)

	for _, agentInfo := range simple.Agents {
		// 次の時刻のRouteを計算
		nextRoute := simple.CalcNextRoute(agentInfo)

		nextControlAgent := &api.Agent{
			Id:    agentInfo.Id,
			Type:  agentInfo.Type,
			Route: nextRoute,
		}
		// Agent追加
		nextControlAgents = append(nextControlAgents, nextControlAgent)
		// 自エリアにいる場合、次のルートを計算する
		/*if IsAgentInArea(agentInfo.Route.Position, simple.Area.ControlArea) {

			// 次の時刻のRouteを計算
			nextRoute := simple.CalcNextRoute(agentInfo, simple.SameAreaAgents)

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
