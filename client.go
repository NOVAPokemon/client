package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/NOVAPokemon/utils"
	"github.com/NOVAPokemon/utils/clients"
	"github.com/NOVAPokemon/utils/notifications"
	"github.com/NOVAPokemon/utils/pokemons"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	configFilename           = "configs.json"
	maxNotificationsBuffered = 10
)

type NovaPokemonClient struct {
	Username string
	Password string

	config *utils.ClientConfig

	authClient          *clients.AuthClient
	battlesClient       *clients.BattleLobbyClient
	tradesClient        *clients.TradeLobbyClient
	notificationsClient *clients.NotificationClient
	trainersClient      *clients.TrainersClient
	storeClient         *clients.StoreClient
	generatorClient     *clients.GeneratorClient
	locationClient      *clients.LocationClient
	gymsClient          *clients.GymClient

	notificationsChannel chan *utils.Notification
	operationsChannel    chan Operation

	emitFinish    chan struct{}
	receiveFinish chan bool
}

var httpCLient = &http.Client{}

func (c *NovaPokemonClient) init() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal("error loading configs")
	}

	c.config = config

	c.notificationsChannel = make(chan *utils.Notification, maxNotificationsBuffered)
	c.operationsChannel = make(chan Operation)

	c.emitFinish = make(chan struct{})
	c.receiveFinish = make(chan bool)

	c.authClient = clients.NewAuthClient(fmt.Sprintf("%s:%d", utils.Host, utils.AuthenticationPort))
	c.battlesClient = &clients.BattleLobbyClient{
		BattlesAddr: fmt.Sprintf("%s:%d", utils.Host, utils.BattlesPort),
	}
	c.tradesClient = clients.NewTradesClient(fmt.Sprintf("%s:%d", utils.Host, utils.TradesPort), c.config.TradeConfig)
	c.notificationsClient = clients.NewNotificationClient(fmt.Sprintf("%s:%d", utils.Host, utils.NotificationsPort), c.notificationsChannel)
	c.trainersClient = clients.NewTrainersClient(fmt.Sprintf("%s:%d", utils.Host, utils.TrainersPort), httpCLient)
	c.storeClient = clients.NewStoreClient(fmt.Sprintf("%s:%d", utils.Host, utils.StorePort))
	c.generatorClient = clients.NewGeneratorClient(fmt.Sprintf("%s:%d", utils.Host, utils.GeneratorPort))
	c.locationClient = clients.NewLocationClient(fmt.Sprintf("%s:%d", utils.Host, utils.LocationPort), c.config.LocationConfig)
	c.gymsClient = clients.NewGymClient(fmt.Sprintf("%s:%d", utils.Host, utils.GymPort), httpCLient)
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
	go c.locationClient.StartLocationUpdates(c.authClient.AuthToken)
}

func (c *NovaPokemonClient) MainLoopAuto() {
	defer c.validateStatsTokens()
	defer c.validateItemTokens()
	defer c.validatePokemonTokens()
	const waitTime = 2 * time.Second
	waitNotificationsTimer := time.NewTimer(waitTime)
	autoClient := NewTrainerSim()
	for {
		select {
		case notification := <-c.notificationsChannel:
			c.HandleNotifications(notification)
		case <-waitNotificationsTimer.C:
			nextOp := autoClient.GetNextOperation(
				c.trainersClient.TrainerStatsClaims,
				c.trainersClient.PokemonClaims,
				c.trainersClient.ItemsClaims)
			exit, err := c.TestOperation(nextOp)
			if err != nil {
				log.Error(err)
				continue
			} else if exit {
				return
			}
		}
		c.validateStatsTokens()
		c.validatePokemonTokens()
		c.validateItemTokens()
		waitNotificationsTimer.Reset(waitTime)
	}
}

func (c *NovaPokemonClient) MainLoopCLI() {
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
				"%c - raid closest gym\n"+
				"%c - exit\n",
			QueueCmd, ChallengeCmd, TradeCmd, StoreCmd, CatchCmd, RaidCmd, ExitCmd)

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
		c.validateStatsTokens()
		c.validatePokemonTokens()
		c.validateItemTokens()
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
		c.operationsChannel <- Operation([]rune(strings.TrimSpace(command))[0])
	}
}

func (c *NovaPokemonClient) TestOperation(operation Operation) (bool, error) {
	switch operation {
	case ChallengeCmd:
		return false, c.startAutoChallenge()
	case QueueCmd:
		return false, c.StartAutoBattleQueue()
	case TradeCmd:
		return false, c.startAutoTrade()
	case StoreCmd:
		return false, c.BuyRandomItem()
	case CatchCmd:
		return false, c.CatchWildPokemon()
	case RaidCmd:
		return false, c.StartLookForNearbyRaid(30 * time.Second)
	case ExitCmd:
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
	statsToken, itemsToken, err := c.storeClient.BuyItem(randomItem.Name, c.authClient.AuthToken, c.trainersClient.TrainerStatsToken)
	if err != nil {
		return err
	}

	err = c.trainersClient.SetItemsToken(itemsToken)

	if err != nil {
		return err
	}

	return c.trainersClient.SetTrainerStatsToken(statsToken)
}

func (c *NovaPokemonClient) CatchWildPokemon() error {
	caught, responseHeader, err := c.generatorClient.CatchWildPokemon(c.authClient.AuthToken, c.trainersClient.ItemsToken)
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

	pokemonsToUse, pokemonTkns, err := c.getPokemonsForBattle(c.config.BattleConfig.PokemonsPerBattle)

	if err != nil {
		return err
	}

	conn, channels, err := c.battlesClient.ChallengePlayerToBattle(c.authClient.AuthToken,
		pokemonTkns,
		c.trainersClient.TrainerStatsToken,
		c.trainersClient.ItemsToken,
		otherPlayer)

	if err != nil {
		return err
	}

	return autoManageBattle(c.trainersClient, conn, *channels, pokemonsToUse)
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
	pokemonsToUse, pokemonTkns, err := c.getPokemonsForBattle(c.config.BattleConfig.PokemonsPerBattle)

	if err != nil {
		return err
	}

	conn, channels, err := c.battlesClient.AcceptChallenge(c.authClient.AuthToken,
		pokemonTkns,
		c.trainersClient.TrainerStatsToken,
		c.trainersClient.ItemsToken,
		battleId)

	if err != nil {
		return err
	}

	return autoManageBattle(c.trainersClient, conn, *channels, pokemonsToUse)
}

func (c *NovaPokemonClient) StartAutoBattleQueue() error {

	pokemonsToUse, pokemonTkns, err := c.getPokemonsForBattle(c.config.BattleConfig.PokemonsPerBattle)

	if err != nil {
		return err
	}

	conn, channels, err := c.battlesClient.QueueForBattle(c.authClient.AuthToken,
		pokemonTkns,
		c.trainersClient.TrainerStatsToken,
		c.trainersClient.ItemsToken)

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

func (c *NovaPokemonClient) StartLookForNearbyRaid(timeout time.Duration) error {

	pokemonsToUse, pokemonTkns, err := c.getPokemonsForBattle(c.config.RaidConfig.PokemonsPerRaid)

	if err != nil {
		return err
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		<-ticker.C
		gyms := c.locationClient.Gyms
		for i := 0; i < len(gyms); i++ {
			gym, err := c.gymsClient.GetGymInfo(gyms[i].Name)
			if err != nil {
				return err
			}

			if gym.RaidBoss == nil || gym.RaidBoss.HP == 0 {
				log.Info("Raidboss was nil or had no hp")
				continue
			}

			log.Info("ongoing raid :", gym.RaidForming)
			if !gym.RaidForming {
				log.Info("Creating a new raid...")
				if err = c.gymsClient.CreateRaid(gym.Name); err != nil {
					log.Error(err)
					return err
				}
			}
			log.Info("Dialing raids...")
			conn, channels, err := c.gymsClient.EnterRaid(c.authClient.AuthToken, pokemonTkns, c.trainersClient.TrainerStatsToken, c.trainersClient.ItemsToken, gym.Name)
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
	}
}

// HELPER FUNCTIONS

func (c *NovaPokemonClient) getPokemonsForBattle(nr int) (map[string]*pokemons.Pokemon, []string, error) {

	var pokemonTkns = make([]string, nr)
	var pokemonMap = make(map[string]*pokemons.Pokemon, nr)

	i := 0
	for _, tkn := range c.trainersClient.PokemonClaims {

		if tkn.Pokemon.HP == 0 {
			continue
		}

		pokemonId := tkn.Pokemon.Id.Hex()
		if i == nr {
			break
		}
		pokemonTkns[i] = c.trainersClient.PokemonTokens[pokemonId]
		aux := c.trainersClient.PokemonClaims[pokemonId].Pokemon // make a copy of pokemons
		pokemonMap[pokemonId] = &aux
		i++
	}
	if i < nr {
		return nil, nil, errors.New("not enough alive pokemons to battle")
	}

	return pokemonMap, pokemonTkns, nil
}

func (c *NovaPokemonClient) validateItemTokens() {
	if valid, err := c.trainersClient.VerifyItems(c.Username, c.trainersClient.ItemsClaims.ItemsHash, c.authClient.AuthToken); err != nil {
		log.Fatal(err)
	} else if !*valid {
		log.Fatal("ended up with wrong items")
	} else {
		log.Info("New item tokens are correct")
	}
}

func (c *NovaPokemonClient) validateStatsTokens() {
	if valid, err := c.trainersClient.VerifyTrainerStats(c.Username, c.trainersClient.TrainerStatsClaims.TrainerHash, c.authClient.AuthToken); err != nil {
		log.Fatal(err)
	} else if !*valid {
		log.Fatal("ended up with wrong stats token")
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
		log.Fatal(err)
	} else if !*valid {
		log.Fatal("ended up with wrong pokemons")
	} else {
		log.Info("New pokemon tokens are correct")
	}
}

func loadConfig() (*utils.ClientConfig, error) {
	fileData, err := ioutil.ReadFile(configFilename)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var clientConfig utils.ClientConfig
	err = json.Unmarshal(fileData, &clientConfig)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.Infof("Loaded battles client config: %+v", clientConfig.BattleConfig)
	log.Infof("Loaded trades client config: %+v", clientConfig.TradeConfig)
	log.Infof("Loaded gym client config: %+v", clientConfig.RaidConfig)
	log.Infof("Loaded location client config: %+v", clientConfig.LocationConfig)

	return &clientConfig, nil
}
