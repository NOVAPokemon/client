package main

import (
	"bufio"
	"fmt"
	"github.com/NOVAPokemon/client/notifications"
	"github.com/NOVAPokemon/utils"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

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

func registerHandlers(client *notifications.NotificationClient) {
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
