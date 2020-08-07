module github.com/NOVAPokemon/client

go 1.13

require (
	github.com/NOVAPokemon/utils v0.0.62
	github.com/gorilla/websocket v1.4.2
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	go.mongodb.org/mongo-driver v1.3.1
)

replace github.com/NOVAPokemon/utils v0.0.62 => ../utils
