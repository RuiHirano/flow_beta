module github.com/synerex/synerex_alpha/cli/simulation/simulator

require (
	github.com/RuiHirano/flow_beta/cli/proto v0.0.0-00010101000000-000000000000
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/mtfelian/golang-socketio v0.0.0-20181017124241-8d8ec6f9bb4c
	github.com/mtfelian/synced v0.0.0-20180626092057-b82cebd56589 // indirect
	github.com/sirupsen/logrus v1.1.1 // indirect
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.2 // indirect
	google.golang.org/grpc v1.53.0
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0
)

replace github.com/RuiHirano/flow_beta/cli/proto => ./proto

go 1.13
