package main

import (
	"bytes"
	"fmt"
	"github.com/NOVAPokemon/utils"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

const hostAddr = "localhost:8002"

func main() {

	jar, err := cookiejar.New(nil)

	if err != nil {
		log.Error(err)
		return
	}


	lobbyClient := NewBattleLobbyClient(hostAddr, utils.Trainer{}, nil)
	battles := GetAvailableLobbies(lobbyClient)

	log.Infof("Available Lobbies: %+v", battles)

	if len(battles) == 0 {

		// login
		_, err = login(jar, "trainer1", "qwe")
		if err != nil {
			log.Fatal(err)
		}
		lobbyClient = NewBattleLobbyClient(hostAddr, utils.Trainer{}, jar)

		// create new lobby
		CreateBattleLobby(lobbyClient)

	} else {

		// login
		_, err = login(jar, "trainer2", "qwe")
		if err != nil {
			log.Fatal(err)
		}
		lobbyClient = NewBattleLobbyClient(hostAddr, utils.Trainer{}, jar)

		// join client
		lobby := battles[0]
		log.Infof("Joining lobby %s", lobby)
		JoinBattleLobby(lobbyClient, lobby.Id)
	}

}

func login(jar *cookiejar.Jar, username string, password string) (*http.Response, error) {

	httpClient := &http.Client{
		Jar: jar,
	}

	//TODO remove these hardcoded credentials

	jsonStr := []byte(fmt.Sprintf(`{"username": "%s", "password": "%s"}`, username, password))
	req, err := http.NewRequest("POST", "http://localhost:8001/login", bytes.NewBuffer(jsonStr))

	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)

	u2, _ := url.Parse("http://localhost:8001/login")
	for _, cookie := range jar.Cookies(u2) {
		log.Info(cookie)
	}

	return resp, err
}
