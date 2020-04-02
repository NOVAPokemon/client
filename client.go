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

func (c *NovaPokemonClient) init() {

	c.jar, _ = cookiejar.New(nil)

	c.authClient = &clients.AuthClient{
		Jar: c.jar,
	}

	c.battlesClient = &clients.BattleLobbyClient{
		BattlesAddr: fmt.Sprintf("%s:%d", utils.Host, utils.BattlesPort),
		Jar:         c.jar,
	}

	c.tradesClient = &clients.TradeLobbyClient{
		TradesAddr: fmt.Sprintf("%s:%d", utils.Host, utils.TradesPort),
		Jar:        c.jar,
	}

	notificationsChan := make(chan *utils.Notification)
	addr := fmt.Sprintf("%s:%d", utils.Host, utils.NotificationsPort)
	c.notificationsClient = clients.NewNotificationClient(addr, c.jar, notificationsChan)
	c.trainersClient = clients.NewTrainersClient(fmt.Sprintf("%s:%d", utils.Host, utils.TrainersPort), c.jar)

}

func (c *NovaPokemonClient) StartTradeWithPlayer(playerId string) {
	trades := c.tradesClient.GetAvailableLobbies()
	log.Infof("Available Lobbies: %+v", trades)

	if len(trades) == 0 {
		lobbyId := c.tradesClient.CreateTradeLobby(playerId)
		log.Info(lobbyId)
		c.tradesClient.JoinTradeLobby(lobbyId)
	} else {
		return

		//lobby := trades[0]
		//log.Infof("Joining lobby %s", lobby)
		//c.tradesClient.JoinTradeLobby(lobby.Id)
	}
}

func (c *NovaPokemonClient) RegisterAndGetTokens() error {
	err := c.authClient.Register(c.Username, c.Password)

	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func (c *NovaPokemonClient) LoginAndGetTokens() error {
	err := c.authClient.LoginWithUsernameAndPassword(c.Username, c.Password)

	if err != nil {
		log.Error(err)
		return err
	}

	err = c.authClient.GetInitialTokens(c.Username)

	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func (c *NovaPokemonClient) LoginAndStartAutoBattleQueue() error {
	err := c.LoginAndGetTokens()

	if err != nil {
		log.Error(err)
		return err
	}

	for ; ; {
		channels := c.battlesClient.QueueForBattle()
		err := autoManageBattle(c, channels)
		if err != nil {
			log.Error(err)
		}
	}
}

func (c *NovaPokemonClient) LoginAndChallegePlayer(otherPlayer string) error {
	err := c.LoginAndGetTokens()

	if err != nil {
		log.Error(err)
		return err
	}

	c.battlesClient.ChallengePlayerToBattle(otherPlayer)

	return nil
}

func LoginAndAcceptChallenges(c *NovaPokemonClient) {
	err := c.LoginAndGetTokens()

	if err != nil {
		log.Error(err)
		return
	}

	go c.notificationsClient.ListenToNotifications()
	err = waitForBattleChallenges(c)

	if err != nil {
		log.Error(err)
		return
	}
}

func (c *NovaPokemonClient) WantingTrade(notification *utils.Notification) error {
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

	time.Sleep(10 * time.Second)

	c.tradesClient.JoinTradeLobby(&lobbyId)
	return nil

}

// Notification Handlers

func (c *NovaPokemonClient) StartListeningToNotifications() {
	for ; ; {
		channels := c.battlesClient.QueueForBattle()
		err := autoManageBattle(c, channels)
		if err != nil {
			log.Error(err)
		}
	}
}

func (c *NovaPokemonClient) ParseReceivedNotifications() {
	for {
		select {
		case notification := <-c.notificationsChannel:
			switch notification.Type {
			case notifications.WantsToTrade:
				c.WantingTrade(notification)
			}
		}
	}
}

func waitForBattleChallenges(c *NovaPokemonClient) error {

	for ; ; {
		notification := <-c.notificationsClient.NotificationsChannel
		switch notification.Type {
		case notifications.ChallengeToBattle:
			log.Info("I was challenged to a battle")
			battleId, err := primitive.ObjectIDFromHex(string(notification.Content))
			if err != nil {
				log.Error(err)
				return err
			}
			c.battlesClient.AcceptChallenge(battleId)
		}
	}

}
