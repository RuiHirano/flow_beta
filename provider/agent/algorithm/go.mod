module github.com/synerex/synerex_alpha/provider/agent/algorithm

require (
	github.com/RuiHirano/flow_beta/api v0.0.0-20201206142946-eadf02ed07a1
	github.com/RuiHirano/flow_beta/util v0.0.0-20201206142946-eadf02ed07a1
	github.com/RuiHirano/rvo2-go/src/rvosimulator v0.0.0-20200118052731-21c801eb6c10
	github.com/paulmach/orb v0.1.5 // indirect
	google.golang.org/grpc v1.53.0 // indirect

)

replace (
	github.com/RuiHirano/flow_beta/api => ./../../api
	github.com/RuiHirano/flow_beta/util => ./../../util
)

go 1.13
