package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/NOVAPokemon/utils"
	"github.com/NOVAPokemon/utils/clients"
	"github.com/NOVAPokemon/utils/notifications"
	"github.com/NOVAPokemon/utils/websockets/battles"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"math/rand"
	"os"
	"strings"
)

const (
	challengeCmd = 'b'
	queueCmd     = 'q'
	tradeCmd     = 't'
	storeCmd     = 's'
	catchCmd     = 'c'
	exitCmd      = 'e'

	maxNotificationsBuffered = 10
)

type NovaPokemonClient struct {
	Username string
	Password string

	authClient          *clients.AuthClient
	battlesClient       *clients.BattleLobbyClient
	tradesClient        *clients.TradeLobbyClient
	notificationsClient *clients.NotificationClient
	trainersClient      *clients.TrainersClient
	storeClient         *clients.StoreClient
	generatorClient     *clients.GeneratorClient

	notificationsChannel chan *utils.Notification
	operationsChannel    chan rune

	emitFinish    chan struct{}
	receiveFinish chan bool
}

func (c *NovaPokemonClient) init() {
	c.notificationsChannel = make(chan *utils.Notification, maxNotificationsBuffered)
	c.operationsChannel = make(chan rune)

	c.emitFinish = make(chan struct{})
	c.receiveFinish = make(chan bool)

	c.authClient = clients.NewAuthClient(fmt.Sprintf("%s:%d", utils.Host, utils.AuthenticationPort))
	c.battlesClient = &clients.BattleLobbyClient{
		BattlesAddr: fmt.Sprintf("%s:%d", utils.Host, utils.BattlesPort),
	}
	c.tradesClient = clients.NewTradesClient(fmt.Sprintf("%s:%d", utils.Host, utils.TradesPort))
	c.notificationsClient = clients.NewNotificationClient(fmt.Sprintf("%s:%d", utils.Host, utils.NotificationsPort), c.notificationsChannel)
	c.trainersClient = clients.NewTrainersClient(fmt.Sprintf("%s:%d", utils.Host, utils.TrainersPort))
	c.storeClient = clients.NewStoreClient(fmt.Sprintf("%s:%d", utils.Host, utils.StorePort))
	c.generatorClient = clients.NewGeneratorClient(fmt.Sprintf("%s:%d", utils.Host, utils.GeneratorPort))
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

func (c *NovaPokemonClient) StartListeningToNotifications() {
	go c.notificationsClient.ListenToNotifications(c.authClient.AuthToken, c.emitFinish, c.receiveFinish)
}

func (c *NovaPokemonClient) StartUpdatingLocation() {
	go c.trainersClient.StartLocationUpdates(c.authClient.AuthToken)
}

func (c *NovaPokemonClient) MainLoop() {
	defer c.validateStatsTokens()
	defer c.validateItemTokens()
	defer c.validatePokemonTokens()
	go c.ReadOperation()
	for {
		fmt.Printf(
			"%c - queue for battle\n"+
				"%c - auto challenge\n"+
				"%c - auto trade\n"+
				"%c - buy random item\n"+
				"%c - try to catch pokemon\n"+
				"%c - exit\n",
			queueCmd, challengeCmd, tradeCmd, storeCmd, catchCmd, exitCmd)

		select {
		case notification := <-c.notificationsChannel:
			c.HandleNotifications(notification)
		case operation := <-c.operationsChannel:
			exit, err := c.TestOperation(operation)
			if err != nil {
				log.Error(err)
				continue
			} else if exit {
				return
			}
		}
	}
}

func (c *NovaPokemonClient) ReadOperation() {
	for {
		reader := bufio.NewReader(os.Stdin)
		command, err := reader.ReadString('\n')
		if err != nil {
			log.Error(err)
			return
		}

		c.operationsChannel <- []rune(strings.TrimSpace(command))[0]
	}
}

func (c *NovaPokemonClient) TestOperation(operation rune) (bool, error) {
	switch operation {
	case challengeCmd:
		return false, c.startAutoChallenge()
	case queueCmd:
		return false, c.StartAutoBattleQueue()
	case tradeCmd:
		return false, c.startAutoTrade()
	case storeCmd:
		return false, c.BuyRandomItem()
	case catchCmd:
		return false, c.CatchWildPokemon()
	case exitCmd:
		return true, nil
	default:
		return false, errors.New("invalid command")
	}
}

func (c *NovaPokemonClient) HandleNotifications(notification *utils.Notification) {
	switch notification.Type {
	case notifications.WantsToTrade:
		err := c.WantingTrade(notification)
		if err != nil {
			log.Error(err)
		}
	case notifications.ChallengeToBattle:
		err := c.handleChallengeNotification(notification)
		if err != nil {
			log.Error(err)
		}
		return
	}
}

func (c *NovaPokemonClient) BuyRandomItem() error {
	items, err := c.storeClient.GetItems(c.authClient.AuthToken)
	if err != nil {
		return err
	}

	randomItem := items[rand.Intn(len(items))]
	itemsToken, err := c.storeClient.BuyItem(randomItem.Name, c.authClient.AuthToken, c.trainersClient.TrainerStatsToken)
	if err != nil {
		return err
	}

	return c.trainersClient.SetItemsToken(itemsToken)
}

func (c *NovaPokemonClient) CatchWildPokemon() error {
	caught, responseHeader, err := c.generatorClient.CatchWildPokemon(c.authClient.AuthToken)
	if err != nil {
		return err
	}
	if !caught {
		log.Info("pokemon got away")
		return nil
	}

	return c.trainersClient.AppendPokemonToken(responseHeader)
}

func (c *NovaPokemonClient) Finish() {
	log.Warn("Finishing client...")
	close(c.emitFinish)

	<-c.receiveFinish
}

func (c *NovaPokemonClient) startAutoChallenge() error {
	trainers, err := c.notificationsClient.GetOthersListening(c.authClient.AuthToken)
	if err != nil {
		return err
	}

	if len(trainers) == 0 {
		log.Warn("No one to challenge")
		return nil
	} else {
		log.Infof("got %d trainers", len(trainers))
	}

	challengePlayer := trainers[rand.Intn(len(trainers))]
	log.Info("Will trade with ", challengePlayer)

	return c.ChallengePlayer(challengePlayer)
}

func (c *NovaPokemonClient) ChallengePlayer(otherPlayer string) error {

	pokemonsForBattle := c.getPokemonsForBattle()
	conn, channels, err := c.battlesClient.ChallengePlayerToBattle(c.authClient.AuthToken, pokemonsForBattle, c.trainersClient.TrainerStatsToken, otherPlayer)

	if err != nil {
		return err
	}

	return autoManageBattle(c.trainersClient, conn, *channels, pokemonsForBattle)
}

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
	conn, channels, err := c.battlesClient.AcceptChallenge(c.authClient.AuthToken, pokemonsForBattle, c.trainersClient.TrainerStatsToken, battleId)

	if err != nil {
		return err
	}

	return autoManageBattle(c.trainersClient, conn, *channels, pokemonsForBattle)
}

// HELPER FUNCTIONS

func (c *NovaPokemonClient) StartAutoBattleQueue() error {

	pokemonsToUse := c.getPokemonsForBattle()
	conn, channels, err := c.battlesClient.QueueForBattle(c.authClient.AuthToken, pokemonsToUse, c.trainersClient.TrainerStatsToken)

	if err != nil {
		log.Error(err)
		return err
	}
	err = autoManageBattle(c.trainersClient, conn, *channels, pokemonsToUse)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func (c *NovaPokemonClient) getPokemonsForBattle() []string {

	var pokemonTkns = make([]string, battles.PokemonsPerBattle)

	i := 0
	for _, tkn := range c.trainersClient.PokemonTokens {
		if i == battles.PokemonsPerBattle {
			break
		}
		pokemonTkns[i] = tkn
		i++
	}
	return pokemonTkns
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

func (c *NovaPokemonClient) validateStatsTokens() {
	if valid, err := c.trainersClient.VerifyTrainerStats(c.Username, c.trainersClient.TrainerStatsClaims.TrainerHash, c.authClient.AuthToken); err != nil {
		log.Error(err)
	} else if !*valid {
		log.Error("ended up with wrong stats token")
	} else {
		log.Info("New stats token is correct")
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
