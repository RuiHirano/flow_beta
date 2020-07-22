module test-provider

go 1.12

require (
	github.com/RuiHirano/synerex_simulation_beta/api v0.0.0-00010101000000-000000000000 // indirect
	github.com/synerex/synerex_api v0.3.1
	github.com/synerex/synerex_proto v0.1.6
	github.com/synerex/synerex_sxutil v0.4.10
)

replace github.com/RuiHirano/synerex_simulation_beta/api => ./../../api
