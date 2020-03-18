package main

import (
	"github.com/NOVAPokemon/utils"
	log "github.com/sirupsen/logrus"
)

func main() {

	var hostAddr = "localhost:8002"
	lobbyClient := NewBattleLobbyClient(hostAddr, utils.Trainer{})

	battles := GetAvailableLobbies(lobbyClient)
	log.Infof("Available Lobbies: %+v", battles)

	print(len(battles))

	if len(battles) == 0 {
		CreateBattleLobby(lobbyClient)
	} else {
		var lobby utils.Lobby = battles[0]
		log.Infof("Joining lobby %s", lobby)
		JoinBattleLobby(lobbyClient, lobby.Id)
	}

}