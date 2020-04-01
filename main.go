package main

import (
	log "github.com/sirupsen/logrus"
	"net/url"
)

func main() {
	client := NovaPokemonClient{
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

	trainers, err := client.trainersClient.ListTrainers()
	if err != nil {
		log.Error(err)
		return
	}

	for _, trainer := range *trainers {
		log.Info(trainer.Username)
	}
}

func testTrades(client *NovaPokemonClient) {
	trades := client.tradesClient.GetAvailableLobbies()
	log.Infof("Available Lobbies: %+v", trades)

	if len(trades) == 0 {
		client.tradesClient.CreateTradeLobby()
	} else {
		lobby := trades[0]
		log.Infof("Joining lobby %s", lobby)
		client.tradesClient.JoinTradeLobby(lobby.Id)
	}
}
