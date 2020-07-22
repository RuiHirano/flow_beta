package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"

	"github.com/go-yaml/yaml"
)

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
	Control   []Coord
	Duplicate []Coord
}

type Coord struct {
	Latitude  float64
	Longitude float64
}

type Option struct {
	FileName        string
	AreaCoords      []Coord
	DevideSquareNum int
	DuplicateRate   float64
}

// vis-monitor
func NewVisMonitorService(area Area) Resource {
	name := "worker" + strconv.Itoa(area.Id)
	monitorName := "vis-monitor" + strconv.Itoa(area.Id)
	service := Resource{
		ApiVersion: "v1",
		Kind:       "Service",
		Metadata:   Metadata{Name: monitorName},
		Spec: Spec{
			Selector: Selector{App: name},
			Ports: []Port{
				{
					Name:       "http",
					Port:       80,
					TargetPort: 9500,
				},
			},
			Type: "NodePort",
		},
	}
	return service
}

// worker
func NewWorkerService(area Area) Resource {
	name := "worker" + strconv.Itoa(area.Id)
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

func NewWorker(area Area) Resource {
	name := "worker" + strconv.Itoa(area.Id)
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
							Value: "SynerexServer" + strconv.Itoa(area.Id),
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
							Value: "WorkerProvider" + strconv.Itoa(area.Id),
						},
					},
					Ports: []Port{{ContainerPort: 9980}},
				},
				{
					Name:            "agent-provider",
					Image:           "synerex-simulation/agent-provider:latest",
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
							Name:  "AREA",
							Value: convertAreaToJson(area),
						},
						{
							Name:  "PROVIDER_NAME",
							Value: "AgentProvider" + strconv.Itoa(area.Id),
						},
					},
				},
				{
					Name:            "visualization-provider",
					Image:           "synerex-simulation/visualization-provider:latest",
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
							Name:  "VIS_ADDRESS",
							Value: ":9500",
						},
						{
							Name:  "PROVIDER_NAME",
							Value: "VisProvider" + strconv.Itoa(area.Id),
						},
					},
				},
			},
		},
	}
	return worker
}

// master
func NewMasterService() Resource {
	service := Resource{
		ApiVersion: "v1",
		Kind:       "Service",
		Metadata:   Metadata{Name: "master"},
		Spec: Spec{
			Selector: Selector{App: "master"},
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
				{
					Name:       "master-provider",
					Port:       800,
					TargetPort: 9990,
				},
			},
		},
	}
	return service
}

func NewMaster() Resource {
	master := Resource{
		ApiVersion: "v1",
		Kind:       "Pod",
		Metadata: Metadata{
			Name:   "master",
			Labels: Label{App: "master"},
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
					},
					Ports: []Port{{ContainerPort: 10000}},
				},
				{
					Name:            "master-provider",
					Image:           "synerex-simulation/master-provider:latest",
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
							Name:  "PORT",
							Value: "9990",
						},
					},
					Ports: []Port{{ContainerPort: 9990}},
				},
			},
		},
	}
	return master
}

// simulator
func NewSimulatorService() Resource {
	service := Resource{
		ApiVersion: "v1",
		Kind:       "Service",
		Metadata:   Metadata{Name: "simulator"},
		Spec: Spec{
			Selector: Selector{App: "simulator"},
			Ports: []Port{
				{
					Name: "http",
					Port: 8000,
				},
			},
			Type: "NodePort",
		},
	}
	return service
}

func NewSimulator() Resource {
	simulator := Resource{
		ApiVersion: "v1",
		Kind:       "Pod",
		Metadata: Metadata{
			Name:   "simulator",
			Labels: Label{App: "simulator"},
		},
		Spec: Spec{
			Containers: []Container{
				{
					Name:            "simulator",
					Image:           "synerex-simulation/simulator:latest",
					ImagePullPolicy: "Never",
					Stdin:           true,
					Tty:             true,
					Env: []Env{
						{
							Name:  "MASTER_ADDRESS",
							Value: "http://master:800",
						},
					},
					Ports: []Port{{ContainerPort: 8000}},
				},
			},
		},
	}
	return simulator
}

// gateway
func NewGateway(neiPair []int) Resource {
	worker1Name := "worker" + strconv.Itoa(neiPair[0])
	worker2Name := "worker" + strconv.Itoa(neiPair[1])
	gatewayName := "gateway" + strconv.Itoa(neiPair[0]) + strconv.Itoa(neiPair[1])
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
							Value: "GatewayProvider" + strconv.Itoa(neiPair[0]) + strconv.Itoa(neiPair[1]),
						},
					},
					Ports: []Port{{ContainerPort: 9980}},
				},
			},
		},
	}
	return gateway
}

func convertAreaToJson(area Area) string {
	id := area.Id
	duplicateText := `[`
	controlText := `[`
	for i, ctl := range area.Control {
		ctlText := fmt.Sprintf(`{"latitude":%v, "longitude":%v}`, ctl.Latitude, ctl.Longitude)
		//fmt.Printf("ctl %v\n", ctlText)
		if i == len(area.Control)-1 { // 最後は,をつけない
			controlText += ctlText
		} else {
			controlText += ctlText + ","
		}
	}
	for i, dpl := range area.Duplicate {
		dplText := fmt.Sprintf(`{"latitude":%v, "longitude":%v}`, dpl.Latitude, dpl.Longitude)
		//fmt.Printf("dpl %v\n", dplText)
		if i == len(area.Duplicate)-1 { // 最後は,をつけない
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

func main() {

	option := Option{
		FileName: "higashiyama-25.yaml",
		AreaCoords: []Coord{
			{Longitude: 136.971626, Latitude: 35.161499},
			{Longitude: 136.971626, Latitude: 35.152210},
			{Longitude: 136.989379, Latitude: 35.152210},
			{Longitude: 136.989379, Latitude: 35.161499},
		},
		DevideSquareNum: 5,   // 2*2 = 4 areas
		DuplicateRate:   0.1, // 10% of each area
	}

	rsrcs := createData(option)
	//fmt.Printf("rsrcs: %v\n", rsrcs)

	// write yaml
	fileName := option.FileName
	for _, rsrc := range rsrcs {
		err := WriteOnFile(fileName, rsrc)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func createData(option Option) []Resource {
	rsrcs := []Resource{
		NewSimulatorService(),
		NewSimulator(),
		NewMasterService(),
		NewMaster(),
	}
	areas, neighbors := AreaDivider(option.AreaCoords, option.DevideSquareNum, option.DuplicateRate)
	//fmt.Printf("areas: %v\n", areas)

	for _, area := range areas {
		rsrcs = append(rsrcs, NewVisMonitorService(area))
		rsrcs = append(rsrcs, NewWorkerService(area))
		rsrcs = append(rsrcs, NewWorker(area))
	}

	for _, neiPair := range neighbors {
		rsrcs = append(rsrcs, NewGateway(neiPair))
	}

	return rsrcs
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

func AreaDivider(areaCoords []Coord, divideSquareNum int, duplicateRate float64) ([]Area, [][]int) {

	neighbors := [][]int{}
	areas := []Area{}

	maxLat, maxLon, minLat, minLon := GetCoordRange(areaCoords)
	for i := 0; i < divideSquareNum; i++ { // 横方向
		leftlon := (maxLon-minLon)*float64(i)/float64(divideSquareNum) + minLon
		rightlon := (maxLon-minLon)*(float64(i)+1)/float64(divideSquareNum) + minLon

		for k := 0; k < divideSquareNum; k++ { // 縦方向
			bottomlat := (maxLat-minLat)*float64(k)/float64(divideSquareNum) + minLat
			toplat := (maxLat-minLat)*(float64(k)+1)/float64(divideSquareNum) + minLat
			id, _ := strconv.Atoi(strconv.Itoa(i+1) + strconv.Itoa(k+1))
			area := Area{
				Id: id,
				Control: []Coord{
					{Longitude: leftlon, Latitude: toplat},
					{Longitude: leftlon, Latitude: bottomlat},
					{Longitude: rightlon, Latitude: bottomlat},
					{Longitude: rightlon, Latitude: toplat},
				},
				Duplicate: []Coord{
					{Longitude: leftlon - (rightlon-leftlon)*duplicateRate, Latitude: toplat + (toplat-bottomlat)*duplicateRate},
					{Longitude: leftlon - (rightlon-leftlon)*duplicateRate, Latitude: bottomlat - (toplat-bottomlat)*duplicateRate},
					{Longitude: rightlon + (rightlon-leftlon)*duplicateRate, Latitude: bottomlat - (toplat-bottomlat)*duplicateRate},
					{Longitude: rightlon + (rightlon-leftlon)*duplicateRate, Latitude: toplat + (toplat-bottomlat)*duplicateRate},
				},
			}
			areas = append(areas, area)

			// add neighbors
			if i+1+1 <= divideSquareNum {
				id2, _ := strconv.Atoi(strconv.Itoa(i+1+1) + strconv.Itoa(k+1))
				neighbors = append(neighbors, []int{id, id2})
			}
			if k+1+1 <= divideSquareNum {
				id3, _ := strconv.Atoi(strconv.Itoa(i+1) + strconv.Itoa(k+1+1))
				neighbors = append(neighbors, []int{id, id3})
			}

		}
	}
	for _, nei := range neighbors {
		fmt.Printf("neighbor: %v\n", nei)
	}

	return areas, neighbors

}

func GetCoordRange(coords []Coord) (float64, float64, float64, float64) {
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
