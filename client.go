package main

import (
	"encoding/json"
	"fmt"
	"github.com/NOVAPokemon/utils"
	"github.com/NOVAPokemon/utils/clients"
	"github.com/NOVAPokemon/utils/notifications"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http/cookiejar"
	"time"
)

type NovaPokemonClient struct {
	Username string
	Password string

	authClient          *clients.AuthClient
	battlesClient       *clients.BattleLobbyClient
	tradesClient        *clients.TradeLobbyClient
	notificationsClient *clients.NotificationClient
	trainersClient      *clients.TrainersClient
	// storeClient *store.StoreClient // TODO

	notificationsChannel chan *utils.Notification

	jar *cookiejar.Jar
}

func (client *NovaPokemonClient) init() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Error(err)
		return
	}
	client.jar = jar

	client.notificationsChannel = make(chan *utils.Notification)

	client.authClient = &clients.AuthClient{
		Jar: client.jar,
	}

	client.battlesClient = &clients.BattleLobbyClient{
		BattlesAddr: fmt.Sprintf("%s:%d", utils.Host, utils.BattlesPort),
		Jar:         client.jar,
	}

	client.tradesClient = clients.NewTradesClient(fmt.Sprintf("%s:%d", utils.Host, utils.TradesPort), client.jar)
	client.notificationsClient = clients.NewNotificationClient(fmt.Sprintf("%s:%d", utils.Host, utils.NotificationsPort), client.jar, client.notificationsChannel)
	client.trainersClient = clients.NewTrainersClient(fmt.Sprintf("%s:%d", utils.Host, utils.TrainersPort), client.jar)

	go client.ParseReceivedNotifications()
}

func (client *NovaPokemonClient) StartAutoClient(username string, password string) {
	client.authClient.Register(client.Username, client.Password)
}

func (client *NovaPokemonClient) StartTradeWithPlayer(playerId string) {
	trades := client.tradesClient.GetAvailableLobbies()
	log.Infof("Available Lobbies: %+v", trades)

	if len(trades) == 0 {
		lobbyId := client.tradesClient.CreateTradeLobby(playerId)
		log.Info(lobbyId)
		client.tradesClient.JoinTradeLobby(lobbyId)
	} else {
		return

		//lobby := trades[0]
		//log.Infof("Joining lobby %s", lobby)
		//client.tradesClient.JoinTradeLobby(lobby.Id)
	}
}

func (client *NovaPokemonClient) Register() {
	client.authClient.Register(client.Username, client.Password)
}

func (client *NovaPokemonClient) Login() {
	client.authClient.LoginWithUsernameAndPassword(client.Username, client.Password)
}

func (client *NovaPokemonClient) GetAllTokens() error {
	return client.authClient.GetInitialTokens(client.Username)
}

func (client *NovaPokemonClient) StartListeningToNotifications() {
	go client.notificationsClient.ListenToNotifications()
}

func (client *NovaPokemonClient) ParseReceivedNotifications() {
	for {
		select {
			case notification := <- client.notificationsChannel:
				switch notification.Type {
				case notifications.WantsToTrade:
					client.WantingTrade(notification)
				}
		}
	}
}

// Notification Handlers

func (client *NovaPokemonClient) WantingTrade(notification *utils.Notification) error {
	var content notifications.WantsToTradeContent
	err := json.Unmarshal(notification.Content, &content)
	if err != nil {
		log.Error(err)
		return err
	}

	lobbyId, err := primitive.ObjectIDFromHex(content.LobbyId)
	if err != nil {
		log.Error(err)
		return err
	}

	time.Sleep(10*time.Second)

	client.tradesClient.JoinTradeLobby(&lobbyId)
	return nil
}
