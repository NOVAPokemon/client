module github.com/NOVAPokemon/client

go 1.13

require (
	github.com/NOVAPokemon/authentication v0.0.7
	github.com/NOVAPokemon/notifications v0.0.1
	github.com/NOVAPokemon/trades v0.0.0-20200331133704-e178969784d1
	github.com/NOVAPokemon/trainers v0.0.1
	github.com/NOVAPokemon/utils v0.0.62
	github.com/gorilla/websocket v1.4.2
	github.com/sirupsen/logrus v1.5.0
	go.mongodb.org/mongo-driver v1.3.1
)

replace (
	github.com/NOVAPokemon/authentication v0.0.7 => ../authentication
	github.com/NOVAPokemon/notifications v0.0.1 => ../notifications
	github.com/NOVAPokemon/trainers v0.0.1 => ../trainers
	github.com/NOVAPokemon/utils v0.0.62 => ../utils
)
