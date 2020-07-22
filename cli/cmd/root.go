package cmd

import (
	//"flag"
	//"encoding/json"
	//"flag"
	"bytes"
	//"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	gosocketio "github.com/mtfelian/golang-socketio"

	"github.com/spf13/cobra"
)

var sioClients []*gosocketio.Client

var sioClient *gosocketio.Client
var Providers []string

var (
	sender *Sender
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "simulator",
	Short: "Synergic Exchange command launcher",
	Long: `Synergic Exchange command launcher
For example:

se is a CLI launcher for Synergic Exchange.

   se run all
   se status   // show status of provider/servers
`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		//		fmt.Println(err)
		os.Exit(1)
	}
}

type Sender struct {
	ServerAddress string
}

func NewSender(serverAddress string) *Sender {
	s := &Sender{
		ServerAddress: serverAddress,
	}
	return s
}

func (s *Sender) Get(data []byte, path string) ([]byte, error) {
	request, err := http.NewRequest("GET", s.ServerAddress+path, bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Error1 occur..., backend-calucator startup?")
		return nil, err
	}
	request.Header.Add("Content-Type", "application/json")
	// 送信
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Printf("Error2 occur..., backend-calucator startup?")
		return nil, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("Error3 occur..., backend-calucator startup?")
		return nil, err
	}
	return body, nil
}

func (s *Sender) Post(data []byte, path string) ([]byte, error) {
	request, err := http.NewRequest("POST", s.ServerAddress+path, bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Error1 occur..., backend-calucator startup?")
		return nil, err
	}
	request.Header.Add("Content-Type", "application/json")
	// 送信
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Printf("Err2 occur..., backend-calucator startup?")
		return nil, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("Err3 occur..., backend-calucator startup?")
		return nil, err
	}
	return body, nil
}

func init() {

	masterProviderAddr := os.Getenv("MASTER_ADDRESS")
	if masterProviderAddr == "" {
		masterProviderAddr = "http://localhost:9990"
	}
	sender = NewSender(masterProviderAddr)

	time.Sleep(200 * time.Millisecond)
}
