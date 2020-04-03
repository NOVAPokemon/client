package main

import (
	"encoding/json"
	"fmt"
	"github.com/NOVAPokemon/utils"
	"github.com/NOVAPokemon/utils/clients"
	"github.com/NOVAPokemon/utils/notifications"
	"github.com/NOVAPokemon/utils/websockets/battles"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
}

func (c *NovaPokemonClient) init() {

	c.authClient = &clients.AuthClient{
	}

	c.battlesClient = &clients.BattleLobbyClient{
		BattlesAddr: fmt.Sprintf("%s:%d", utils.Host, utils.BattlesPort),
	}

	c.tradesClient = &clients.TradeLobbyClient{
		TradesAddr: fmt.Sprintf("%s:%d", utils.Host, utils.TradesPort),
	}

	notificationsChan := make(chan *utils.Notification)
	addr := fmt.Sprintf("%s:%d", utils.Host, utils.NotificationsPort)
	c.notificationsClient = clients.NewNotificationClient(addr, notificationsChan)
	c.trainersClient = clients.NewTrainersClient(fmt.Sprintf("%s:%d", utils.Host, utils.TrainersPort))

}

func (c *NovaPokemonClient) StartTradeWithPlayer(playerId string) {
	trades := c.tradesClient.GetAvailableLobbies()
	log.Infof("Available Lobbies: %+v", trades)

	if len(trades) == 0 {
		lobbyId := c.tradesClient.CreateTradeLobby(playerId, c.authClient.AuthToken, c.trainersClient.ItemsToken)
		log.Info(lobbyId)
		c.tradesClient.JoinTradeLobby(lobbyId, c.authClient.AuthToken, c.trainersClient.ItemsToken)
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

	err = c.trainersClient.GetAllTrainerTokens(c.Username)

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
		channels := c.battlesClient.QueueForBattle(c.authClient.AuthToken, c.getPokemonsForBattle())
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

	c.battlesClient.ChallengePlayerToBattle(c.authClient.AuthToken, c.getPokemonsForBattle(), otherPlayer)

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

	c.tradesClient.JoinTradeLobby(&lobbyId, c.authClient.AuthToken, c.trainersClient.ItemsToken)
	return nil

}

func (c *NovaPokemonClient) StartAutoBattleQueue() {
	for ; ; {
		channels := c.battlesClient.QueueForBattle(c.authClient.AuthToken, c.getPokemonsForBattle())
		err := autoManageBattle(c, channels)
		if err != nil {
			log.Error(err)
		}
	}
}

// Notification Handlers

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

func (c *NovaPokemonClient) getPokemonsForBattle() map[string]string {
	var pokemonTkns = make(map[string]string, battles.PokemonsPerBattle)
	for k, v := range c.trainersClient.PokemonTokens {
		pokemonTkns[k] = v
		if len(pokemonTkns) == battles.PokemonsPerBattle {
			break
		}
	}
	return pokemonTkns
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
			c.battlesClient.AcceptChallenge(c.authClient.AuthToken, c.getPokemonsForBattle(), battleId)
		}
	}
}
