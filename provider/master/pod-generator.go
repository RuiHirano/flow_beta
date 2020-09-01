package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"strconv"

	api "github.com/RuiHirano/flow_beta/api"
	"github.com/go-yaml/yaml"
)

//////////////////////////////////
////////// Pod Generator ////////
//////////////////////////////////

type PodGenerator struct {
	RsrcMap  map[string][]Resource
	ImageID  string
	ImageVer string
}

func NewPodGenerator(imageID string, imageVer string) *PodGenerator {
	pg := &PodGenerator{
		RsrcMap:  make(map[string][]Resource),
		ImageID:  imageID,
		ImageVer: imageVer,
	}
	return pg
}

func (pg *PodGenerator) applyWorker(area *api.Area, neighborIds []string) error {
	fmt.Printf("applying WorkerPod... %v\n", area.Id)
	areaid := strconv.FormatUint(area.Id, 10)
	rsrcs := []Resource{
		pg.NewWorkerService(areaid),
		pg.NewWorker(area, neighborIds),
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

func (pg *PodGenerator) NewWorker(area *api.Area, neighborIds []string) Resource {
	areaid := strconv.FormatUint(area.Id, 10)
	workerName := "worker" + areaid

	containers := []Container{
		{
			Name:            "synerex-nodeserv",
			Image:           fmt.Sprintf("%s/synerex-nodeserv:%s", pg.ImageID, pg.ImageVer),
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
			Image:           fmt.Sprintf("%s/synerex-server:%s", pg.ImageID, pg.ImageVer),
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
			Image:           fmt.Sprintf("%s/worker-provider:%s", pg.ImageID, pg.ImageVer),
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
					Value: "WorkerProvider" + areaid,
				},
			},
		},
		{
			Name:            "agent-provider",
			Image:           fmt.Sprintf("%s/agent-provider:%s", pg.ImageID, pg.ImageVer),
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
					Value: "AgentProvider" + areaid,
				},
			},
		},
	}
	for i, nid := range neighborIds {
		neighborWorkerName := "worker" + nid
		containers = append(containers, Container{
			Name:            "gateway-provider" + strconv.Itoa(i),
			Image:           fmt.Sprintf("%s/gateway-provider:%s", pg.ImageID, pg.ImageVer),
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
					Value: "GatewayProvider" + areaid + nid,
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
