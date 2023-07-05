module github.com/synerex/synerex_alpha/provider/scenario-provider

require (
	github.com/RuiHirano/flow_beta/api v0.0.0-00010101000000-000000000000
	github.com/RuiHirano/flow_beta/util v0.0.0-00010101000000-000000000000
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
	github.com/synerex/synerex_api v0.4.1
	github.com/synerex/synerex_sxutil v0.4.12
	google.golang.org/grpc v1.53.0 // indirect
)

replace (
	github.com/RuiHirano/flow_beta/api => ./../../api
	github.com/RuiHirano/flow_beta/util => ./../../util
)

go 1.13
