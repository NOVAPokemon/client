package main

import (
	"encoding/json"
	"fmt"
	"github.com/NOVAPokemon/utils"
	"github.com/NOVAPokemon/utils/clients"
	"github.com/NOVAPokemon/utils/notifications"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"math/rand"
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

const MaxNotifications = 20

func (c *NovaPokemonClient) init() {

	c.jar, _ = cookiejar.New(nil)

	c.authClient = &clients.AuthClient{
		Jar: c.jar,
	}

	c.battlesClient = &clients.BattleLobbyClient{
		BattlesAddr: fmt.Sprintf("%s:%d", utils.Host, utils.BattlesPort),
		Jar:         c.jar,
	}

	c.tradesClient = clients.NewTradesClient(fmt.Sprintf("%s:%d", utils.Host, utils.TradesPort), c.jar)

	c.notificationsChannel = make(chan *utils.Notification)
	addr := fmt.Sprintf("%s:%d", utils.Host, utils.NotificationsPort)

	c.notificationsClient = clients.NewNotificationClient(addr, c.jar, c.notificationsChannel)

	c.trainersClient = clients.NewTrainersClient(fmt.Sprintf("%s:%d", utils.Host, utils.TrainersPort), c.jar)
}

func (c *NovaPokemonClient) StartTradeWithPlayer(playerId string) {
	lobbyId := c.tradesClient.CreateTradeLobby(playerId)
	log.Info("Created lobby ", lobbyId)
	c.tradesClient.JoinTradeLobby(lobbyId)
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

func (c *NovaPokemonClient) LoginAndChallegePlayer(otherPlayer string) error {
	err := c.LoginAndGetTokens()

	if err != nil {
		log.Error(err)
		return err
	}

	c.battlesClient.ChallengePlayerToBattle(otherPlayer)

	return nil
}

func (c *NovaPokemonClient) LoginAndAcceptChallenges() {
	err := c.LoginAndGetTokens()

	if err != nil {
		log.Error(err)
		return
	}

	go c.notificationsClient.ListenToNotifications()
	err = c.waitForBattleChallenges()

	if err != nil {
		log.Error(err)
		return
	}
}

func (c *NovaPokemonClient) StartListeningToNotifications() {
	go c.notificationsClient.ListenToNotifications()
}

func (c *NovaPokemonClient) ParseReceivedNotifications() {
	waitDuration := 5 * time.Second

	waitForNotificationTimer := time.NewTimer(waitDuration)

	for {
		<-waitForNotificationTimer.C
		select {
		case notification := <-c.notificationsChannel:
			switch notification.Type {
			case notifications.WantsToTrade:
				err := c.WantingTrade(notification)
				if err != nil {
					log.Error(err)
				}
			}
		default:
			err := c.startAutoTrade()
			if err != nil {
				return
			}
		}
		waitForNotificationTimer.Reset(waitDuration)
	}
}

func (c *NovaPokemonClient) startAutoTrade() error {
	trainers, err := c.notificationsClient.GetOthersListening(c.Username)
	if err != nil {
		log.Error(err)
		return err
	}

	if len(trainers) == 0 {
		log.Warn("No one to trade with")
		return nil
	} else {
		log.Infof("got %d trainers", len(trainers))
	}

	tradeWith := trainers[rand.Intn(len(trainers))]
	log.Info("Will trade with ", tradeWith)
	c.StartTradeWithPlayer(tradeWith)

	return nil
}

// Notification Handlers

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

	c.tradesClient.JoinTradeLobby(&lobbyId)
	return nil

}

func (c *NovaPokemonClient) waitForBattleChallenges() error {

	for {
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
