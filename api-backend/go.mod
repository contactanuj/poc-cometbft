module poc-cometbft/api-backend

go 1.22.0

require (
	github.com/CosmWasm/wasmd v0.53.4
	github.com/cometbft/cometbft v0.38.12
	github.com/cosmos/cosmos-sdk v0.50.14
	github.com/go-chi/chi/v5 v5.1.0
	google.golang.org/grpc v1.64.0
	poc-cometbft/mychain v0.0.0
)

replace (
	poc-cometbft/mychain => ../mychain
	github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/99designs/keyring => github.com/cosmos/keyring v1.2.0
	github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/gin-gonic/gin => github.com/gin-gonic/gin v1.9.1
)
