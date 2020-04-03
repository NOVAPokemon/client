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
	"math/rand"
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

	c.tradesClient = clients.NewTradesClient(fmt.Sprintf("%s:%d", utils.Host, utils.TradesPort))

	c.notificationsChannel = make(chan *utils.Notification)
	addr := fmt.Sprintf("%s:%d", utils.Host, utils.NotificationsPort)

	c.notificationsClient = clients.NewNotificationClient(addr, c.notificationsChannel)

	c.trainersClient = clients.NewTrainersClient(fmt.Sprintf("%s:%d", utils.Host, utils.TrainersPort))
}

func (c *NovaPokemonClient) StartTradeWithPlayer(playerId string) {
	lobbyId := c.tradesClient.CreateTradeLobby(playerId, c.authClient.AuthToken, c.trainersClient.ItemsToken)
	log.Info("Created lobby ", lobbyId)

	newItemTokens := c.tradesClient.JoinTradeLobby(lobbyId, c.authClient.AuthToken, c.trainersClient.ItemsToken)

	if err := c.trainersClient.SetItemsToken(*newItemTokens); err != nil {
		log.Error(err)
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

	err = c.trainersClient.GetAllTrainerTokens(c.Username, c.authClient.AuthToken)

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

func (c *NovaPokemonClient) LoginAndAcceptChallenges() {
	err := c.LoginAndGetTokens()

	if err != nil {
		log.Error(err)
		return
	}

	go c.notificationsClient.ListenToNotifications(c.authClient.AuthToken)
	err = c.waitForBattleChallenges()

	if err != nil {
		log.Error(err)
		return
	}
}

func (c *NovaPokemonClient) StartListeningToNotifications() {
	go c.notificationsClient.ListenToNotifications(c.authClient.AuthToken)
}

func (c *NovaPokemonClient) ParseReceivedNotifications() {
	waitDuration := 10 * time.Second

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
				return
			}
		default:
			_ = c.startAutoTrade()
			return
		}
		waitForNotificationTimer.Reset(waitDuration)
	}
}

func (c *NovaPokemonClient) startAutoTrade() error {
	trainers, err := c.notificationsClient.GetOthersListening(c.Username, c.authClient.AuthToken)
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

func (c *NovaPokemonClient) waitForBattleChallenges() error {

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
