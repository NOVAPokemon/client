package main

import (
	log "github.com/sirupsen/logrus"
	"net/url"
)

func main() {
	client := NovaPokemonClient{}
	client.init()

	username := requestUsername()
	password := requestPassword()

	client.authClient.LoginWithUsernameAndPassword(username, password)

	err := client.authClient.GetInitialTokens(username)
	if err != nil {
		log.Error(err)
		return
	}

	for _, cookie := range client.jar.Cookies(&url.URL{
		Scheme:     "http",
		Host:       "localhost",
	}) {
		log.Info(cookie)
	}

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