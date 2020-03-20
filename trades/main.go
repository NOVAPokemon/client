package main

import (
	"bytes"
	"github.com/NOVAPokemon/utils"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

func main() {
	jar, err := cookiejar.New(nil)

	if err != nil {
		log.Error(err)
		return
	}

	login(jar)

	var hostAddr = "localhost:8003"
	lobbyClient := NewTradeLobbyClient(hostAddr, utils.Trainer{})

	battles := GetAvailableLobbies(lobbyClient, jar)
	log.Infof("Available Lobbies: %+v", battles)

	print(len(battles))

	if len(battles) == 0 {
		CreateTradeLobby(lobbyClient)
	} else {
		var lobby utils.Lobby = battles[0]
		log.Infof("Joining lobby %s", lobby)
		JoinTradeLobby(lobbyClient, lobby.Id)
	}
}

func login(jar *cookiejar.Jar) {

	httpClient := &http.Client{
		Jar: jar,
	}

	jsonStr := []byte(`{"username": "teste", "password": "ola"}`)
	req, err := http.NewRequest("POST", "http://localhost:8001/login", bytes.NewBuffer(jsonStr))

	if err != nil {
		log.Error(err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)

	log.Info(resp)

	if err != nil {
		log.Error(err)
		return
	}

	u2, _ := url.Parse("http://localhost:8001/login")
	for _, cookie := range jar.Cookies(u2) {
		log.Info(cookie)
	}
}
