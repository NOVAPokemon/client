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

	client := &NotificationClient{
		notificationHandlers: map[string]utils.NotificationHandler{},
		jar:                  jar,
	}

	registerHandlers(client)

	hostAddr := fmt.Sprintf("%s:%d", utils.Host, utils.NotificationsPort)
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
