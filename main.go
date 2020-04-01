package main

import (
	log "github.com/sirupsen/logrus"
	"net/url"
)

func main() {
	client := &NovaPokemonClient{
		Username: requestUsername(),
		Password: requestPassword(),
	}
	client.init()

	client.Login()
	err := client.GetAllTokens()
	if err != nil {
		log.Error(err)
		return
	}

	for _, cookie := range client.jar.Cookies(&url.URL{
		Scheme: "http",
		Host:   "localhost",
	}) {
		log.Info(cookie)
	}

	client.StartListeningToNotifications()

	client.StartTradeWithPlayer(requestUsername())
}


func testTrainers(client *NovaPokemonClient) {
	trainers, err := client.trainersClient.ListTrainers()
	if err != nil {
		log.Error(err)
		return
	}

	for _, trainer := range *trainers {
		log.Info(trainer.Username)
	}
}