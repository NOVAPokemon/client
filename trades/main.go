package main

import (
	"fmt"
	clientUtils "github.com/NOVAPokemon/client/utils"
	"github.com/NOVAPokemon/utils"
	log "github.com/sirupsen/logrus"
	"net/http/cookiejar"
)

func main() {
	jar, err := cookiejar.New(nil)

	if err != nil {
		log.Error(err)
		return
	}

	clientUtils.Login(jar)

	var hostAddr = fmt.Sprintf("%s:%d", utils.Host, utils.TradesPort)
	lobbyClient := NewTradeLobbyClient(hostAddr, utils.Trainer{}, jar)

	battles := GetAvailableLobbies(lobbyClient)
	log.Infof("Available Lobbies: %+v", battles)

	print(len(battles))

	if len(battles) == 0 {
		CreateTradeLobby(lobbyClient)
	} else {
		lobby := battles[0]
		log.Infof("Joining lobby %s", lobby)
		JoinTradeLobby(lobbyClient, lobby.Id)
	}
}
