module poc-cometbft/mychain

go 1.22.0

require (
	cosmossdk.io/api v0.7.5
	cosmossdk.io/client/v2 v2.0.0-beta.1
	cosmossdk.io/core v0.11.0
	cosmossdk.io/depinject v1.0.0-alpha.4
	cosmossdk.io/errors v1.0.1
	cosmossdk.io/log v1.3.1
	cosmossdk.io/math v1.3.0
	cosmossdk.io/store v1.1.0
	cosmossdk.io/tools/confix v0.1.1
	cosmossdk.io/x/circuit v0.1.1
	cosmossdk.io/x/evidence v0.1.1
	cosmossdk.io/x/feegrant v0.1.1
	cosmossdk.io/x/upgrade v0.1.4
	github.com/CosmWasm/wasmd v0.53.4
	github.com/CosmWasm/wasmvm/v2 v2.1.4
	github.com/cometbft/cometbft v0.38.12
	github.com/cosmos/cosmos-db v1.0.2
	github.com/cosmos/cosmos-sdk v0.50.14
	github.com/cosmos/gogoproto v1.7.0
	github.com/cosmos/ibc-go/modules/capability v1.0.1
	github.com/cosmos/ibc-go/v8 v8.4.0
	github.com/gorilla/mux v1.8.1
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/spf13/cast v1.6.0
	github.com/spf13/cobra v1.8.0
	github.com/spf13/viper v1.18.2
	github.com/stretchr/testify v1.9.0
	google.golang.org/genproto/googleapis/api v0.0.0-20240318140521-94a12d6c2237
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.34.2
)

replace (
	// Use goleveldb instead of cleveldb for simpler builds
	github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	// Fix keyring dependency
	github.com/99designs/keyring => github.com/cosmos/keyring v1.2.0
	// Fix jwt-go
	github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt/v4 v4.5.0
	// Fix gin security issue
	github.com/gin-gonic/gin => github.com/gin-gonic/gin v1.9.1
)
