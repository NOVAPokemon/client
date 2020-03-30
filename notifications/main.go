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

	client := &NotificationClient{
		notificationHandlers: map[string]utils.NotificationHandler{},
		jar:                  jar,
	}

	registerHandlers(client)

	var hostAddr = "localhost:8010"
	client.ListenToNotifications(hostAddr)
}

func registerHandlers(client *NotificationClient) {
	err := client.RegisterHandler("ANY",
		func(notification utils.Notification) error {
			log.Infof("%+v", notification)
			return nil
		})

	if err != nil {
		log.Error(err)
		return
	}
}

func login(jar *cookiejar.Jar) {

	httpClient := &http.Client{
		Jar: jar,
	}

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
