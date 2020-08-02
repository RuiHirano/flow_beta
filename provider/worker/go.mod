module github.com/synerex/synerex_alpha/provider/scenario-provider

require (
	github.com/RuiHirano/flow_beta/api v0.0.0-00010101000000-000000000000 // indirect
	github.com/RuiHirano/flow_beta/util v0.0.0-00010101000000-000000000000 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/google/logger v1.0.1 // indirect
	github.com/google/uuid v1.1.1
	github.com/labstack/echo v3.3.10+incompatible // indirect
	github.com/labstack/gommon v0.3.0 // indirect
	github.com/mtfelian/golang-socketio v1.5.2
	github.com/paulmach/orb v0.1.5 // indirect
	google.golang.org/grpc v1.30.0
)

replace (
	github.com/RuiHirano/flow_beta/api => ./../../api
	github.com/RuiHirano/flow_beta/util => ./../../util
)

go 1.13
