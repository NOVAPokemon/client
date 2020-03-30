package main

import (
	"encoding/json"
	"errors"
	notifications "github.com/NOVAPokemon/notifications/exported"
	"github.com/NOVAPokemon/utils"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

type NotificationClient struct {
	notificationHandlers map[string]utils.NotificationHandler
	jar                  *cookiejar.Jar
}

func (client *NotificationClient) ListenToNotifications(addr string) {
	u := url.URL{Scheme: "ws", Host: addr, Path: notifications.SubscribeNotificationPath}

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		Jar:              client.jar,
	}

	c, _, err := dialer.Dial(u.String(), nil)
	defer c.Close()

	if err != nil {
		log.Fatal(err)
		return
	}

	client.readNotifications(c)

	log.Info("Stopped listening to notifications...")
}

func (client *NotificationClient) RegisterHandler(notificationType string, handler utils.NotificationHandler) error {
	oldHandler := client.notificationHandlers[notificationType]
	if oldHandler != nil {
		return errors.New("notification already handled")
	}

	client.notificationHandlers[notificationType] = handler
	log.Infof("Registered handler for type: %s", notificationType)

	return nil
}

func (client *NotificationClient) readNotifications(conn *websocket.Conn) {
	for {
		_, jsonBytes, err := conn.ReadMessage()

		if err != nil {
			log.Error(err)
			return
		}

		var notification utils.Notification
		err = json.Unmarshal(jsonBytes, &notification)

		if err != nil {
			log.Error(err)
			return
		}

		handler := client.notificationHandlers[notification.Type]

		if handler == nil {
			log.Errorf("cant handle notification type: %s", notification.Type)
			log.Errorf("%+v", client.notificationHandlers)
			continue
		}

		err = handler(notification)

		if err != nil {
			log.Error(err)
		}

		log.Debugf("Received %s from the websocket", notification.Content)
	}
}
