module github.com/NOVAPokemon/client

go 1.13

require (
	github.com/NOVAPokemon/utils v0.0.62
	github.com/gorilla/websocket v1.4.2
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	go.mongodb.org/mongo-driver v1.3.1
)

replace (
	github.com/NOVAPokemon/utils v0.0.62 => ../utils
	github.com/bruno-anjos/archimedesHTTPClient v0.0.2 => ./../../bruno-anjos/archimedesHTTPClient
	github.com/bruno-anjos/cloud-edge-deployment v0.0.1 => ../../bruno-anjos/cloud-edge-deployment
	github.com/nm-morais/demmon-client v1.0.0 => ../../nm-morais/demmon-client
    github.com/nm-morais/demmon-common v1.0.0 => ../../nm-morais/demmon-common
    github.com/nm-morais/demmon-exporter v1.0.2 => ../../nm-morais/demmon-exporter
)
