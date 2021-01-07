package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/go-yaml/yaml"

	pb "github.com/RuiHirano/flow_beta/cli/proto"
	"github.com/spf13/cobra"
	"gopkg.in/go-playground/validator.v9"
)

var (
	//geoInfo *geojson.FeatureCollection

	config *Config
)

func init() {

	// configを読み取る
	config, _ = readConfig()
}

type Config struct {
	Area Config_Area `yaml:"area"`
}

type Config_Area struct {
	Coords []*Coord `yaml:"coords"`
}

type Coord struct {
	Latitude  float64 `yaml:"latitude"`
	Longitude float64 `yaml:"longitude"`
}

func readConfig() (*Config, error) {
	var config *Config
	buf, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		fmt.Println(err)
		return config, err
	}
	// []map[string]string のときと使う関数は同じです。
	// いい感じにマッピングしてくれます。
	err = yaml.Unmarshal(buf, &config)
	if err != nil {
		fmt.Println(err)
		return config, err
	}
	fmt.Printf("yaml is %v\n", config)
	return config, nil
}

/////////////////////////////////////////////////
/////////           Stop Command            /////
////////////////////////////////////////////////

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Simulation",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("stop\n")
		request := &pb.StopClockRequest{}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		//response, err := client.RunProcess(ctx, request)
		client.StopClock(ctx, request)
		//sender.Post(nil, "/order/start")
		//sender.Post(nil, "/order/stop")

	},
}

func init() {
	orderCmd.AddCommand(stopCmd)
}

/////////////////////////////////////////////////
/////////           Start Command            /////
////////////////////////////////////////////////

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Simulation",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("start\n")
		request := &pb.StartClockRequest{}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		//response, err := client.RunProcess(ctx, request)
		client.StartClock(ctx, request)
		//sender.Post(nil, "/order/start")

	},
}

func init() {
	orderCmd.AddCommand(startCmd)

}

/////////////////////////////////////////////////
/////////           Set Command            /////
////////////////////////////////////////////////

type AgentOptions struct {
	Num int `validate:"required,min=0,max=100000", json:"num"`
}

type AreaOptions struct {
	SLat string `min=0,max=100", json:"slat"`
	SLon string `min=0,max=200", json:"slon"`
	ELat string `min=0,max=100", json:"elat"`
	ELon string `min=0,max=200", json:"elon"`
}

type ClockOptions struct {
	Time int `validate:"required,min=0" json:"time"`
}

type ConfigOptions struct {
	ConfigName string `validate:"required" json:"config_name"`
}

type InfectionOptions struct {
	Radius float64  `validate:"required,min=0,max=1", json:"radius"` // 半径何m?以内にいる人が接触と判断するか
	Rate float64 `validate:"required,min=0,max=1", json:"rate"`// 何%が感染するか
}

var (
	ao  = &AgentOptions{}
	aro = &AreaOptions{}
	co  = &ClockOptions{}
	coo = &ConfigOptions{}
	io = &InfectionOptions{}
)

var simulatorRequestCmd = &cobra.Command{
	Use:   "inf",
	Short: "order reset",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("simulation request\n")
		//aojson, _ := json.Marshal(ao)
		data, _ := json.Marshal(io)
		request := &pb.SimulatorRequest{
			Type: "SET_PARAM",
			Data: string(data),
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		//response, err := client.RunProcess(ctx, request)
		client.Simulator(ctx, request)
		//sender.Post(aojson, "/order/set/agent")
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return validateParams(*io)
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "order reset",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("reset\n")
		//aojson, _ := json.Marshal(ao)
		request := &pb.ResetRequest{}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		//response, err := client.RunProcess(ctx, request)
		client.Reset(ctx, request)
		//sender.Post(aojson, "/order/set/agent")
	},
}

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Set agent or clock or area",
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Set agent",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("set agent\n")
		//aojson, _ := json.Marshal(ao)
		request := &pb.SetAgentRequest{
			Num: uint64(ao.Num),
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		//response, err := client.RunProcess(ctx, request)
		client.SetAgent(ctx, request)
		//sender.Post(aojson, "/order/set/agent")
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return validateParams(*ao)
	},
}

var areaCmd = &cobra.Command{
	Use:   "area",
	Short: "Set area",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("set area %v\n", aro)
		request := &pb.SetAreaRequest{}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		//response, err := client.RunProcess(ctx, request)
		client.SetArea(ctx, request)
		//arojson, _ := json.Marshal(aro)
		//sender.Post(arojson, "/order/set/area")
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return validateParams(*aro)
	},
}

var clockCmd = &cobra.Command{
	Use:   "clock",
	Short: "Set clock",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("set clock %v\n", co.Time)
		request := &pb.SetClockRequest{
			Time: uint64(co.Time),
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		//response, err := client.RunProcess(ctx, request)
		client.SetClock(ctx, request)
		//cojson, _ := json.Marshal(co)
		//sender.Post(cojson, "/order/set/clock")
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return validateParams(*co)
	},
}
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Set config",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("set config %v\n", coo.ConfigName)
		request := &pb.SetConfigRequest{
			ConfigName: coo.ConfigName,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		client.SetConfig(ctx, request)
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return validateParams(*coo)
	},
}

func init() {
	simulatorRequestCmd.Flags().Float64VarP(&io.Rate, "rate", "p", 0.5, "infect rate (required)")
	simulatorRequestCmd.Flags().Float64VarP(&io.Radius, "radius", "r", 0.00006, "infect radius (required)")
	agentCmd.Flags().IntVarP(&ao.Num, "num", "n", 0, "agent num (required)")
	areaCmd.Flags().StringVarP(&aro.ELat, "elat", "a", "35.161544", "area end latitude (required)")
	areaCmd.Flags().StringVarP(&aro.SLat, "slat", "b", "35.152371", "area start latitude (required)")
	areaCmd.Flags().StringVarP(&aro.ELon, "elon", "c", "136.989860", "area end lonitude (required)")
	areaCmd.Flags().StringVarP(&aro.SLon, "slon", "d", "136.971762", "area start lonitude (required)")
	clockCmd.Flags().IntVarP(&co.Time, "time", "t", 0, "clcok time (required)")
	configCmd.Flags().StringVarP(&coo.ConfigName, "name", "n", "", "config fine name (required)")
	setCmd.AddCommand(agentCmd)
	setCmd.AddCommand(clockCmd)
	setCmd.AddCommand(areaCmd)
	setCmd.AddCommand(configCmd)
	orderCmd.AddCommand(setCmd)
	orderCmd.AddCommand(resetCmd)
	orderCmd.AddCommand(simulatorRequestCmd)  // simulation独自のコマンド
}

/////////////////////////////////////////////////
//////////          Order Command          /////
////////////////////////////////////////////////
var orderCmd = &cobra.Command{
	Use:   "order",
	Short: "Start a provider",
	Long: `Start a provider with options 
For example:
    simulation order start   
	simulation order set-time   
	simulation order set-area   
`,
}

func init() {
	rootCmd.AddCommand(orderCmd)
}

/////////////////////////////////////////////////
//////////            Validation            /////
////////////////////////////////////////////////
var validate = validator.New()

func validateParams(p interface{}) error {

	errs := validate.Struct(p)

	return extractValidationErrors(errs)
}

func extractValidationErrors(err error) error {

	if err != nil {
		var errorText []string
		for _, err := range err.(validator.ValidationErrors) {
			errorText = append(errorText, validationErrorToText(err))
		}
		return fmt.Errorf("Parameter error: %s", strings.Join(errorText, "\n"))
	}

	return nil
}

func validationErrorToText(e validator.FieldError) string {

	f := e.Field()
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", f)
	case "max":
		return fmt.Sprintf("%s cannot be greater than %s", f, e.Param())
	case "min":
		return fmt.Sprintf("%s must be greater than %s", f, e.Param())
	}
	return fmt.Sprintf("%s is not valid %s", e.Field(), e.Value())
}
