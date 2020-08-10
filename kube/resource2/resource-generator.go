package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"

	"github.com/go-yaml/yaml"
)

var (
	imageID  = flag.String("imageID", getImageID(), "Image ID")
	imageVer = flag.String("imageVer", getImageVersion(), "Image Version")
	fileName = flag.String("fileName", getFileName(), "File Name")
)

func getImageID() string {
	env := os.Getenv("IMAGE_ID")
	if env != "" {
		return env
	} else {
		return "docker.pkg.github.com/ruihirano/flow_beta"
	}
}

func getImageVersion() string {
	env := os.Getenv("IMAGE_VER")
	if env != "" {
		return env
	} else {
		return "latest"
	}
}

func getFileName() string {
	env := os.Getenv("FILE_NAME")
	if env != "" {
		return env
	} else {
		return ""
	}
}

func init() {
	flag.Parse()
	log.Printf("Flag Args: ImageID: %s, ImageVersion: %s, FileName: %s", *imageID, *imageVer, *fileName)
	if *fileName == "" {
		log.Printf("Input FileName")
		os.Exit(1)
	}
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
	NodePort      int    `yaml:"nodePort,omitempty"`
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
	Id          int
	Control     []Coord
	Duplicate   []Coord
	NeighborIds []int // neighbor area
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
func NewVisMonitorService() Resource {
	name := "visualization"
	monitorName := "vis-monitor"
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
					NodePort:   30000,
				},
			},
			Type: "NodePort",
		},
	}
	return service
}

// Visualization
func NewVisService() Resource {
	name := "visualization"
	service := Resource{
		ApiVersion: "v1",
		Kind:       "Service",
		Metadata:   Metadata{Name: name},
		Spec: Spec{
			Selector: Selector{App: name},
			Ports: []Port{
				{
					Name:       "synerex",
					Port:       10000,
					TargetPort: 10000,
				},
				{
					Name:       "nodeid",
					Port:       9000,
					TargetPort: 9000,
				},
			},
		},
	}
	return service
}

func NewVis() Resource {
	vis := Resource{
		ApiVersion: "v1",
		Kind:       "Pod",
		Metadata: Metadata{
			Name:   "visualization",
			Labels: Label{App: "visualization"},
		},
		Spec: Spec{
			Containers: []Container{
				{
					Name:            "synerex-nodeserv",
					Image:           fmt.Sprintf("%s/synerex-nodeserv:%s", *imageID, *imageVer),
					ImagePullPolicy: "IfNotPresent",
					Env: []Env{
						{
							Name:  "SX_NODESERV_HOST",
							Value: "visualization",
						},
						{
							Name:  "SX_NODESERV_PORT",
							Value: "9000",
						},
						{
							Name:  "SX_NODESERV_VERSION",
							Value: "false",
						},
						{
							Name:  "SX_NODESERV_VEBOSE",
							Value: "false",
						},
						{
							Name:  "SX_NODESERV_RESTART",
							Value: "false",
						},
					},
					Ports: []Port{{ContainerPort: 9000}},
				},
				{
					Name:            "synerex-server",
					Image:           fmt.Sprintf("%s/synerex-server:%s", *imageID, *imageVer),
					ImagePullPolicy: "IfNotPresent",
					Env: []Env{
						{
							Name:  "SX_NODESERV_HOST",
							Value: "visualization",
						},
						{
							Name:  "SX_NODESERV_PORT",
							Value: "9000",
						},
						{
							Name:  "SX_SERVER_HOST",
							Value: "visualization",
						},
						{
							Name:  "SX_SERVER_PORT",
							Value: "10000",
						},
						{
							Name:  "SX_SERVER_NAME",
							Value: "SynerexServer",
						},
						{
							Name:  "SX_SERVER_METRICS",
							Value: "false",
						},
					},
					Ports: []Port{{ContainerPort: 10000}},
				},
				{
					Name:            "visualization-provider",
					Image:           fmt.Sprintf("%s/visualization-provider:%s", *imageID, *imageVer),
					ImagePullPolicy: "IfNotPresent",
					Env: []Env{
						{
							Name:  "SX_NODESERV_ADDRESS",
							Value: "visualization:9000",
						},
						{
							Name:  "SX_SERVER_ADDRESS",
							Value: "visualization:10000",
						},
						{
							Name:  "SX_MASTER_NODESERV_ADDRESS",
							Value: "master:9000",
						},
						{
							Name:  "SX_MASTER_SERVER_ADDRESS",
							Value: "master:10000",
						},
						{
							Name:  "PROVIDER_NAME",
							Value: "VisProvider",
						},
						{
							Name:  "MONITOR_ADDRESS",
							Value: "visualization:9500",
						},
					},
					Ports: []Port{{ContainerPort: 9500}},
				},
			},
		},
	}
	return vis
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
					Port:       10000,
					TargetPort: 10000,
				},
				{
					Name:       "nodeid",
					Port:       9000,
					TargetPort: 9000,
				},
			},
		},
	}
	return service
}

func NewWorker(area Area) Resource {
	workerName := "worker" + strconv.Itoa(area.Id)

	containers := []Container{
		{
			Name:            "synerex-nodeserv",
			Image:           fmt.Sprintf("%s/synerex-nodeserv:%s", *imageID, *imageVer),
			ImagePullPolicy: "IfNotPresent",
			Env: []Env{
				{
					Name:  "SX_NODESERV_HOST",
					Value: workerName,
				},
				{
					Name:  "SX_NODESERV_PORT",
					Value: "9000",
				},
				{
					Name:  "SX_NODESERV_VERSION",
					Value: "false",
				},
				{
					Name:  "SX_NODESERV_VEBOSE",
					Value: "false",
				},
				{
					Name:  "SX_NODESERV_RESTART",
					Value: "false",
				},
			},
			Ports: []Port{{ContainerPort: 9000}},
		},
		{
			Name:            "synerex-server",
			Image:           fmt.Sprintf("%s/synerex-server:%s", *imageID, *imageVer),
			ImagePullPolicy: "IfNotPresent",
			Env: []Env{
				{
					Name:  "SX_NODESERV_HOST",
					Value: workerName,
				},
				{
					Name:  "SX_NODESERV_PORT",
					Value: "9000",
				},
				{
					Name:  "SX_SERVER_HOST",
					Value: workerName,
				},
				{
					Name:  "SX_SERVER_PORT",
					Value: "10000",
				},
				{
					Name:  "SX_SERVER_NAME",
					Value: "SynerexServer",
				},
				{
					Name:  "SX_SERVER_METRICS",
					Value: "false",
				},
			},
			Ports: []Port{{ContainerPort: 10000}},
		},
		{
			Name:            "worker-provider",
			Image:           fmt.Sprintf("%s/worker-provider:%s", *imageID, *imageVer),
			ImagePullPolicy: "IfNotPresent",
			Env: []Env{
				{
					Name:  "SX_NODESERV_ADDRESS",
					Value: workerName + ":9000",
				},
				{
					Name:  "SX_SERVER_ADDRESS",
					Value: workerName + ":10000",
				},
				{
					Name:  "SX_MASTER_NODESERV_ADDRESS",
					Value: "master:9000",
				},
				{
					Name:  "SX_MASTER_SERVER_ADDRESS",
					Value: "master:10000",
				},
				{
					Name:  "PROVIDER_NAME",
					Value: "WorkerProvider" + strconv.Itoa(area.Id),
				},
			},
		},
		{
			Name:            "agent-provider",
			Image:           fmt.Sprintf("%s/agent-provider:%s", *imageID, *imageVer),
			ImagePullPolicy: "IfNotPresent",
			Env: []Env{
				{
					Name:  "SX_NODESERV_ADDRESS",
					Value: workerName + ":9000",
				},
				{
					Name:  "SX_SERVER_ADDRESS",
					Value: workerName + ":10000",
				},
				{
					Name:  "SX_VIS_SERVER_ADDRESS",
					Value: "visualization:10000",
				},
				{
					Name:  "SX_VIS_NODESERV_ADDRESS",
					Value: "visualization:9000",
				},
				{
					Name:  "AREA_JSON",
					Value: convertAreaToJson(area),
				},
				{
					Name:  "PROVIDER_NAME",
					Value: "AgentProvider" + strconv.Itoa(area.Id),
				},
			},
		},
	}
	for i, nid := range area.NeighborIds {
		neighborWorkerName := "worker" + strconv.Itoa(nid)
		containers = append(containers, Container{
			Name:            "gateway-provider" + strconv.Itoa(i),
			Image:           fmt.Sprintf("%s/gateway-provider:%s", *imageID, *imageVer),
			ImagePullPolicy: "IfNotPresent",
			Env: []Env{
				{
					Name:  "SX_NODESERV_ADDRESS",
					Value: workerName + ":9000",
				},
				{
					Name:  "SX_SERVER_ADDRESS",
					Value: workerName + ":10000",
				},
				{
					Name:  "SX_WORKER_NODESERV_ADDRESS",
					Value: fmt.Sprintf("%s:9000", neighborWorkerName),
				},
				{
					Name:  "SX_WORKER_SERVER_ADDRESS",
					Value: fmt.Sprintf("%s:10000", neighborWorkerName),
				},
				{
					Name:  "PROVIDER_NAME",
					Value: "GatewayProvider" + strconv.Itoa(area.Id) + strconv.Itoa(nid),
				},
			},
		})
	}
	worker := Resource{
		ApiVersion: "v1",
		Kind:       "Pod",
		Metadata: Metadata{
			Name:   workerName,
			Labels: Label{App: workerName},
		},
		Spec: Spec{
			Containers: containers,
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
					Port:       10000,
					TargetPort: 10000,
				},
				{
					Name:       "nodeid",
					Port:       9000,
					TargetPort: 9000,
				},
				{
					Name:       "master-provider",
					Port:       9900,
					TargetPort: 9900,
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
					Name:            "synerex-nodeserv",
					Image:           fmt.Sprintf("%s/synerex-nodeserv:%s", *imageID, *imageVer),
					ImagePullPolicy: "IfNotPresent",
					Env: []Env{
						{
							Name:  "SX_NODESERV_HOST",
							Value: "master",
						},
						{
							Name:  "SX_NODESERV_PORT",
							Value: "9000",
						},
						{
							Name:  "SX_NODESERV_VERSION",
							Value: "false",
						},
						{
							Name:  "SX_NODESERV_VEBOSE",
							Value: "false",
						},
						{
							Name:  "SX_NODESERV_RESTART",
							Value: "false",
						},
					},
					Ports: []Port{{ContainerPort: 9000}},
				},
				{
					Name:            "synerex-server",
					Image:           fmt.Sprintf("%s/synerex-server:%s", *imageID, *imageVer),
					ImagePullPolicy: "IfNotPresent",
					Env: []Env{
						{
							Name:  "SX_NODESERV_HOST",
							Value: "master",
						},
						{
							Name:  "SX_NODESERV_PORT",
							Value: "9000",
						},
						{
							Name:  "SX_SERVER_HOST",
							Value: "master",
						},
						{
							Name:  "SX_SERVER_PORT",
							Value: "10000",
						},
						{
							Name:  "SX_SERVER_NAME",
							Value: "SynerexServer",
						},
						{
							Name:  "SX_SERVER_METRICS",
							Value: "false",
						},
					},
					Ports: []Port{{ContainerPort: 10000}},
				},
				{
					Name:            "master-provider",
					Image:           fmt.Sprintf("%s/master-provider:%s", *imageID, *imageVer),
					ImagePullPolicy: "IfNotPresent",
					Env: []Env{
						{
							Name:  "SX_NODESERV_ADDRESS",
							Value: "master:9000",
						},
						{
							Name:  "SX_SERVER_ADDRESS",
							Value: "master:10000",
						},
						{
							Name:  "CLI_PORT",
							Value: "9900",
						},
						{
							Name:  "PROVIDER_NAME",
							Value: "MasterProvider",
						},
					},
					Ports: []Port{{ContainerPort: 9900}},
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
					Image:           fmt.Sprintf("%s/simulator:%s", *imageID, *imageVer),
					ImagePullPolicy: "IfNotPresent",
					Stdin:           true,
					Tty:             true,
					Env: []Env{
						{
							Name:  "MASTER_ADDRESS",
							Value: "http://master:9900",
						},
					},
					Ports: []Port{{ContainerPort: 8000}},
				},
			},
		},
	}
	return simulator
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
		FileName: *fileName + ".yaml",
		AreaCoords: []Coord{
			{Longitude: 136.971626, Latitude: 35.161499},
			{Longitude: 136.971626, Latitude: 35.152210},
			{Longitude: 136.989379, Latitude: 35.152210},
			{Longitude: 136.989379, Latitude: 35.161499},
		},
		DevideSquareNum: 2,   // 2*2 = 4 areas
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
		NewVisMonitorService(),
		NewVisService(),
		NewVis(),
	}
	areas := AreaDivider(option.AreaCoords, option.DevideSquareNum, option.DuplicateRate)
	//fmt.Printf("areas: %v\n", areas)

	for _, area := range areas {
		//rsrcs = append(rsrcs, NewVisMonitorService(area))
		rsrcs = append(rsrcs, NewWorkerService(area))
		rsrcs = append(rsrcs, NewWorker(area))
		//rsrcs = append(rsrcs, NewDatabase(area))
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

func AreaDivider(areaCoords []Coord, divideSquareNum int, duplicateRate float64) []Area {

	//neighbors := [][]int{}
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

			// add neighbors
			if i+1+1 <= divideSquareNum {
				id2, _ := strconv.Atoi(strconv.Itoa(i+1+1) + strconv.Itoa(k+1))
				area.NeighborIds = append(area.NeighborIds, id2)
			}
			if k+1+1 <= divideSquareNum {
				id3, _ := strconv.Atoi(strconv.Itoa(i+1) + strconv.Itoa(k+1+1))
				area.NeighborIds = append(area.NeighborIds, id3)
			}

			areas = append(areas, area)

		}
	}

	return areas

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
