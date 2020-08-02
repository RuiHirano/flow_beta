module github.com/synerex/synerex_alpha/provider/gateway-provider

require (
	github.com/RuiHirano/flow_beta/api v0.0.0-00010101000000-000000000000 // indirect
	github.com/RuiHirano/flow_beta/util v0.0.0-00010101000000-000000000000 // indirect
	github.com/mtfelian/golang-socketio v1.5.2
	google.golang.org/grpc v1.30.0
)

replace (
	github.com/RuiHirano/flow_beta/api => ./../../api
	github.com/RuiHirano/flow_beta/util => ./../../util
)

go 1.13
