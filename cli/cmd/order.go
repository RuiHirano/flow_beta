package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/go-yaml/yaml"

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
		sender.Post(nil, "/order/stop")

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
		sender.Post(nil, "/order/start")

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

var (
	ao  = &AgentOptions{}
	aro = &AreaOptions{}
	co  = &ClockOptions{}
)

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Set agent or clock or area",
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Set agent",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("set agent\n")
		aojson, _ := json.Marshal(ao)
		sender.Post(aojson, "/order/set/agent")
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
		arojson, _ := json.Marshal(aro)
		sender.Post(arojson, "/order/set/area")
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
		cojson, _ := json.Marshal(co)
		sender.Post(cojson, "/order/set/clock")
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return validateParams(*co)
	},
}

func init() {
	agentCmd.Flags().IntVarP(&ao.Num, "num", "n", 0, "agent num (required)")
	areaCmd.Flags().StringVarP(&aro.ELat, "elat", "a", "35.161544", "area end latitude (required)")
	areaCmd.Flags().StringVarP(&aro.SLat, "slat", "b", "35.152371", "area start latitude (required)")
	areaCmd.Flags().StringVarP(&aro.ELon, "elon", "c", "136.989860", "area end lonitude (required)")
	areaCmd.Flags().StringVarP(&aro.SLon, "slon", "d", "136.971762", "area start lonitude (required)")
	clockCmd.Flags().IntVarP(&co.Time, "time", "t", 0, "clcok time (required)")
	setCmd.AddCommand(agentCmd)
	setCmd.AddCommand(clockCmd)
	setCmd.AddCommand(areaCmd)
	orderCmd.AddCommand(setCmd)
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
