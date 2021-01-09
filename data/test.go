package main

import (
	"log"
	"math/rand"
	"strconv"
	"strings"

	//"encoding/csv"
	"io/ioutil"

	"github.com/jszwec/csvutil"
)

type Link struct {
	LinkID string `csv:"LinkID"`
	StartNodeID string `csv:"StartNodeID"`
	EndNodeID string `csv:"EndNodeID"`
	Type int `csv:"Type"`
}

func getLinkData() map[string]Link{
	var linkData []Link
// バイト列を読み込む
	b, _ := ioutil.ReadFile("./nagoya/link.csv")
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
	b, _ := ioutil.ReadFile("./nagoya/exit.csv")
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
	b, _ := ioutil.ReadFile("./nagoya/node.csv")
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
	b, _ := ioutil.ReadFile("./nagoya/facility.csv")
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
	b, _ := ioutil.ReadFile("./nagoya/hospital.csv")
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

type Coord struct {
	Latitude             float64
	Longitude            float64
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

func createRoute(transitNum int, nodeData map[string]Node, linkData map[string]Link) (Coord, Coord, []Coord){
	// departure
	keys := make([]string, len(nodeData))
	i := 0
	for k := range nodeData {
		keys[i] = k
		i++
	}
	randkey := keys[rand.Intn(len(nodeData))]
	departureNode := nodeData[randkey]
	departure := Coord{Latitude: departureNode.Latitude, Longitude: departureNode.Longitude}
	
	// transitpoints
	transitPoints := []Coord{}
	tgtNode := departureNode
	for i := 0; i < transitNum; i++ {
		log.Print(tgtNode.NodeID)
		nextLink := linkData[getRandomNodeLinkID(tgtNode)]
		nextNode := nodeData[nextLink.StartNodeID]
		if tgtNode.Latitude == nextNode.Latitude && tgtNode.Longitude == nextNode.Longitude{
			nextNode = nodeData[nextLink.EndNodeID]  // 同じNodeに戻らないようにする
		}
		transitPoints = append(transitPoints, Coord{
			Latitude: nextNode.Latitude,
			Longitude: nextNode.Longitude,
		})
		tgtNode = nextNode
	}
	// destination
	destinationLink := linkData[tgtNode.LinkID1]
	destinationNode := nodeData[destinationLink.EndNodeID]
	destination := Coord{Latitude: destinationNode.Latitude, Longitude: destinationNode.Longitude}
	return departure, destination, transitPoints
}

func main(){
	//log.Printf("test")
	linkData := getLinkData()
	//log.Printf("linkData: %v", linkData[0])
	nodeData := getNodeData()
	//log.Printf("nodeData: %v", nodeData[0])
	//facilityData := getFacilityData()
	//log.Printf("facilityData: %v", facilityData[0])
	//hospitalData := getHospitalData()
	//log.Printf("hospitalData: %v", hospitalData[0])
	//exitData := getExitData()
	//log.Printf("exitData: %v", exitData[0])

	departure, destination, transitPoints := createRoute(10, nodeData, linkData)
	log.Printf("departure: %v\n destination: %v\n transitPoint: %v", departure, destination, transitPoints)

}