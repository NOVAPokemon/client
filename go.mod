module github.com/NOVAPokemon/client

go 1.13

require (
	github.com/NOVAPokemon/authentication v0.0.7 // indirect
	github.com/NOVAPokemon/generator v0.0.0-20200408175633-e2bd1b3478fb // indirect
	github.com/NOVAPokemon/store v0.0.0-20200402234902-75f7792046b7 // indirect
	github.com/NOVAPokemon/trainers v0.0.3 // indirect
	github.com/NOVAPokemon/utils v0.0.62
	github.com/gorilla/websocket v1.4.2
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.5.0
	go.mongodb.org/mongo-driver v1.3.1
)

replace github.com/NOVAPokemon/utils v0.0.62 => ../utils
