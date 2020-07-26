module test-provider

go 1.12

require (
	github.com/RuiHirano/flow_beta/api v0.0.0-00010101000000-000000000000 // indirect
	github.com/RuiHirano/flow_beta/util v0.0.0-00010101000000-000000000000 // indirect
	github.com/synerex/synerex_api v0.4.1
	github.com/synerex/synerex_nodeapi v0.5.3
	github.com/synerex/synerex_proto v0.1.8
	github.com/synerex/synerex_sxutil v0.4.12
)

replace (
	github.com/RuiHirano/flow_beta/api => ./../../api
	github.com/RuiHirano/flow_beta/util => ./../../util
)
