package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/NOVAPokemon/utils"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
)

func main() {
	jar, err := cookiejar.New(nil)

	if err != nil {
		log.Error(err)
		return
	}

	login(jar)

	var hostAddr = "localhost:8003"
	lobbyClient := NewTradeLobbyClient(hostAddr, utils.Trainer{}, jar)

	battles := GetAvailableLobbies(lobbyClient)
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

	//TODO remove these hardcoded credentials

	username := requestUsername()
	password := requestPassword()

	jsonStr, err := json.Marshal(utils.UserJSON{Username: username, Password: password})
	if err != nil {
		log.Error(err)
	}
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

func requestUsername() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter username: ")
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func requestPassword() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter password: ")
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}
