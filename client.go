package main

import (
	"fmt"
	"github.com/NOVAPokemon/utils"
	"github.com/NOVAPokemon/utils/clients"
	log "github.com/sirupsen/logrus"
	"net/http/cookiejar"
)

type NovaPokemonClient struct {
	Username string
	Password string

	authClient          *clients.AuthClient
	battlesClient       *clients.BattleLobbyClient
	tradesClient        *clients.TradeLobbyClient
	notificationsClient *clients.NotificationClient
	trainersClient      *clients.TrainersClient
	jar                 *cookiejar.Jar
	// storeClient *store.StoreClient // TODO
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

func (c *NovaPokemonClient) RegisterAndGetTokens() error {
	err := c.authClient.Register(c.Username, c.Password)

	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func LoginAndGetTokens(c *NovaPokemonClient) error {
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

func LoginAndStartAutoBattleQueue(c *NovaPokemonClient) error {
	err := LoginAndGetTokens(c)

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

func LoginAndChallegePlayer(c *NovaPokemonClient, otherPlayer string) error {
	err := LoginAndGetTokens(c)

	if err != nil {
		log.Error(err)
		return err
	}

	c.battlesClient.ChallengePlayerToBattle(otherPlayer)

	return nil
}

func LoginAndAcceptChallenges(c *NovaPokemonClient) {
	err := LoginAndGetTokens(c)

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
