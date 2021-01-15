module github.com/synerex/synerex_alpha/provider/agent-provider

require (
	cloud.google.com/go/firestore v1.4.0 // indirect
	firebase.google.com/go v3.13.0+incompatible // indirect
	github.com/RuiHirano/flow_beta/api v0.0.0-00010101000000-000000000000 // indirect
	github.com/RuiHirano/flow_beta/provider/agent/algorithm v0.0.0-00010101000000-000000000000 // indirect
	github.com/RuiHirano/flow_beta/util v0.0.0-00010101000000-000000000000 // indirect
	github.com/RuiHirano/rvo2-go v1.1.1 // indirect
	github.com/jszwec/csvutil v1.4.0 // indirect
	github.com/paulmach/orb v0.1.5
	google.golang.org/grpc v1.33.2
)

replace (
	github.com/RuiHirano/flow_beta/api => ./../../api
	github.com/RuiHirano/flow_beta/provider/agent/algorithm => ./algorithm
	github.com/RuiHirano/flow_beta/util => ./../../util
)

go 1.13
