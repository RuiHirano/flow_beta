module util

go 1.12

require (
	github.com/synerex/synerex_api v0.4.1
	github.com/synerex/synerex_nodeapi v0.5.3
	github.com/synerex/synerex_sxutil v0.4.12
)

replace (
	github.com/RuiHirano/flow_beta/api => ./../../api
	github.com/RuiHirano/flow_beta/util => ./../../util
)
