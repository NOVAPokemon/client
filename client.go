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
	emitFinish           chan struct{}
	receiveFinish        chan bool
}

func (c *NovaPokemonClient) init() {

	c.authClient = &clients.AuthClient{}

	c.battlesClient = &clients.BattleLobbyClient{
		BattlesAddr: fmt.Sprintf("%s:%d", utils.Host, utils.BattlesPort),
	}

	c.tradesClient = clients.NewTradesClient(fmt.Sprintf("%s:%d", utils.Host, utils.TradesPort))

	c.notificationsChannel = make(chan *utils.Notification)
	c.emitFinish = make(chan struct{})
	c.receiveFinish = make(chan bool)

	addr := fmt.Sprintf("%s:%d", utils.Host, utils.NotificationsPort)

	c.notificationsClient = clients.NewNotificationClient(addr, c.notificationsChannel)

	c.trainersClient = clients.NewTrainersClient(fmt.Sprintf("%s:%d", utils.Host, utils.TrainersPort))
}

func (c *NovaPokemonClient) StartTradeWithPlayer(playerId string) {
	lobbyId := c.tradesClient.CreateTradeLobby(playerId, c.authClient.AuthToken, c.trainersClient.ItemsToken)
	log.Info("Created lobby ", lobbyId)

	c.JoinTradeWithPlayer(lobbyId)
}

func (c *NovaPokemonClient) JoinTradeWithPlayer(lobbyId *primitive.ObjectID) {
	newItemTokens := c.tradesClient.JoinTradeLobby(lobbyId, c.authClient.AuthToken, c.trainersClient.ItemsToken)

	if newItemTokens != nil {
		if err := c.trainersClient.SetItemsToken(*newItemTokens); err != nil {
			log.Error(err)
		}
	}
}

func (c *NovaPokemonClient) RegisterAndGetTokens() error {
	err := c.authClient.Register(c.Username, c.Password)

	if err != nil {
		log.Error(err)
		return err
	}

	return c.LoginAndGetTokens()
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

func (c *NovaPokemonClient) ChallegePlayer(otherPlayer string) error {

	channels, err := c.battlesClient.ChallengePlayerToBattle(c.authClient.AuthToken, c.getPokemonsForBattle(), otherPlayer)

	if err != nil {
		return err
	}

	return autoManageBattle(*channels, c.getPokemonsForBattle())
}

func (c *NovaPokemonClient) LoginAndAcceptChallenges() {
	err := c.LoginAndGetTokens()

	if err != nil {
		log.Error(err)
		return
	}

	c.StartListeningToNotifications()
	err = c.waitForBattleChallenges()

	if err != nil {
		log.Error(err)
		return
	}
}

func (c *NovaPokemonClient) StartListeningToNotifications() {
	go c.notificationsClient.ListenToNotifications(c.authClient.AuthToken, c.emitFinish, c.receiveFinish)
}

func (c *NovaPokemonClient) ParseReceivedNotifications() {
	waitDuration := 10 * time.Second

	waitForNotificationTimer := time.NewTimer(waitDuration)

	defer c.validateItemTokens()
	defer c.validatePokemonTokens()

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
			case notifications.ChallengeToBattle:
				err := c.handleChallengeNotification(notification)
				if err != nil {
					log.Error(err)
				}
				return
			}

		default:
			err := c.startAutoChallenge()
			if err != nil {
				log.Error(err)
			}
			return
		}
		//waitForNotificationTimer.Reset(waitDuration)
	}
}

func (c *NovaPokemonClient) Finish() {
	log.Warn("Finishing client...")
	close(c.emitFinish)

	<-c.receiveFinish
}

// Trades

func (c *NovaPokemonClient) startAutoTrade() error {
	trainers, err := c.notificationsClient.GetOthersListening(c.authClient.AuthToken)
	if err != nil {
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

func (c *NovaPokemonClient) validateItemTokens() {
	if valid, err := c.trainersClient.VerifyItems(c.Username, c.trainersClient.ItemsClaims.ItemsHash, c.authClient.AuthToken); err != nil {
		log.Error(err)
	} else if !*valid {
		log.Error("ended up with wrong items")
	} else {
		log.Info("New item tokens are correct")
	}
}

func (c *NovaPokemonClient) validatePokemonTokens() {

	hashes := make(map[string][]byte, len(c.trainersClient.PokemonClaims))
	for _, tkn := range c.trainersClient.PokemonClaims {
		hashes[tkn.Pokemon.Id.Hex()] = tkn.PokemonHash
	}

	if valid, err := c.trainersClient.VerifyPokemons(c.Username, hashes, c.authClient.AuthToken); err != nil {
		log.Error(err)
	} else if !*valid {
		log.Error("ended up with wrong pokemons")
	} else {
		log.Info("New pokemon tokens are correct")
	}
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

	c.JoinTradeWithPlayer(&lobbyId)

	return nil
}

func (c *NovaPokemonClient) handleChallengeNotification(notification *utils.Notification) error {
	log.Info("I was challenged to a battle")
	battleId, err := primitive.ObjectIDFromHex(string(notification.Content))
	if err != nil {
		log.Error(err)
		return err
	}
	pokemonsForBattle := c.getPokemonsForBattle()
	channels, err := c.battlesClient.AcceptChallenge(c.authClient.AuthToken, pokemonsForBattle, battleId)

	if err != nil {
		return err
	}

	return autoManageBattle(*channels, pokemonsForBattle)
}

// HELPER FUNCTIONS

func (c *NovaPokemonClient) StartAutoBattleQueue() error {

	pokemonsToUse := c.getPokemonsForBattle()

	for i, p := range pokemonsToUse {
		log.Info(i, p)
	}

	channels, err := c.battlesClient.QueueForBattle(c.authClient.AuthToken, pokemonsToUse)

	if err != nil {
		log.Error(err)
		return err
	}
	err = autoManageBattle(*channels, pokemonsToUse)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func (c *NovaPokemonClient) getPokemonsForBattle() []string {

	var pokemonTkns = make([]string, battles.PokemonsPerBattle)

	for i := 0; i < len(c.trainersClient.PokemonTokens) && i < battles.PokemonsPerBattle; i++ {
		pokemonTkns[i] = c.trainersClient.PokemonTokens[i]
	}

	return pokemonTkns
}
