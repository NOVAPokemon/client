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
	authRefreshTime          = 20

	AUTO = "AUTO"
	CLI  = "CLI"
)

type NovaPokemonClient struct {
	Username string
	Password string

	config *utils.ClientConfig

	authClient              *clients.AuthClient
	battlesClient           *clients.BattleLobbyClient
	tradesClient            *clients.TradeLobbyClient
	notificationsClient     *clients.NotificationClient
	trainersClient          *clients.TrainersClient
	storeClient             *clients.StoreClient
	locationClient          *clients.LocationClient
	gymsClient              *clients.GymClient
	microtransacitonsClient *clients.MicrotransactionsClient

	notificationsChannel chan *utils.Notification
	operationsChannel    chan Operation

	emitFinish    chan struct{}
	receiveFinish chan bool
}

var httpCLient = &http.Client{}

func (c *NovaPokemonClient) init() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	c.config = config

	c.notificationsChannel = make(chan *utils.Notification, maxNotificationsBuffered)
	c.operationsChannel = make(chan Operation)

	c.emitFinish = make(chan struct{})
	c.receiveFinish = make(chan bool)

	c.authClient = clients.NewAuthClient()
	c.battlesClient = clients.NewBattlesClient()
	c.tradesClient = clients.NewTradesClient(c.config.TradeConfig)
	c.notificationsClient = clients.NewNotificationClient(c.notificationsChannel)
	c.trainersClient = clients.NewTrainersClient(httpCLient)
	c.storeClient = clients.NewStoreClient()
	c.locationClient = clients.NewLocationClient(c.config.LocationConfig)
	c.gymsClient = clients.NewGymClient(httpCLient)
	c.microtransacitonsClient = clients.NewMicrotransactionsClient()
}

func (c *NovaPokemonClient) StartTradeWithPlayer(playerId string) error {
	lobbyId, serverName, err := c.tradesClient.CreateTradeLobby(playerId, c.authClient.AuthToken, c.trainersClient.ItemsToken)
	if err != nil {
		return wrapStartTradeError(err)
	}

	log.Info("Created lobby ", lobbyId)
	return c.JoinTradeWithPlayer(lobbyId, *serverName)
}

func (c *NovaPokemonClient) JoinTradeWithPlayer(lobbyId *primitive.ObjectID, serverHostname string) error {
	newItemTokens, err := c.tradesClient.JoinTradeLobby(lobbyId, serverHostname, c.authClient.AuthToken,
		c.trainersClient.ItemsToken)
	if err != nil {
		return wrapJoinTradeError(err)
	}

	if newItemTokens != nil {
		if err := c.trainersClient.SetItemsToken(*newItemTokens); err != nil {
			return wrapJoinTradeError(err)
		}
	}

	return nil
}

func (c *NovaPokemonClient) RejectTradeWithPlayer(lobbyId *primitive.ObjectID, serverHostname string) error {
	err := c.tradesClient.RejectTrade(lobbyId, serverHostname, c.authClient.AuthToken, c.trainersClient.ItemsToken)

	return wrapRejectTradeError(err)
}

func (c *NovaPokemonClient) RegisterAndGetTokens() error {
	err := c.authClient.Register(c.Username, c.Password)
	if err != nil {
		return wrapRegisterAndGetTokensError(err)
	}

	err = c.LoginAndGetTokens()
	if err != nil {
		return wrapRegisterAndGetTokensError(err)
	}

	return nil
}

func (c *NovaPokemonClient) LoginAndGetTokens() error {
	err := c.authClient.LoginWithUsernameAndPassword(c.Username, c.Password)
	if err != nil {
		return wrapLoginAndGeTokensError(err)
	}

	err = c.trainersClient.GetAllTrainerTokens(c.Username, c.authClient.AuthToken)
	if err != nil {
		return wrapLoginAndGeTokensError(err)
	}

	return nil
}

func (c *NovaPokemonClient) StartListeningToNotifications() {
	go func() {
		err := c.notificationsClient.ListenToNotifications(c.authClient.AuthToken, c.emitFinish, c.receiveFinish)
		if err != nil {
			log.Error(err)
		}
	}()
}

func (c *NovaPokemonClient) StartUpdatingLocation() {
	go func() {
		err := c.locationClient.StartLocationUpdates(c.authClient.AuthToken, c.trainersClient)
		if err != nil {
			log.Error(err)
		}
	}()
}

func (c *NovaPokemonClient) MainLoopAuto() {
	defer c.validateStatsTokens()
	defer c.validateItemTokens()
	defer c.validatePokemonTokens()

	authTimer := time.NewTimer(authRefreshTime * time.Minute)

	const waitTime = 2 * time.Second
	waitNotificationsTimer := time.NewTimer(waitTime)
	autoClient := NewTrainerSim()
	for {
		select {
		case notification := <-c.notificationsChannel:
			c.HandleNotifications(notification, c.operationsChannel, AUTO)
		case <-waitNotificationsTimer.C:
			nextOp := autoClient.GetNextOperation()
			exit, err := c.TestOperation(nextOp)
			if err != nil {
				log.Error(err)
			} else if exit {
				return
			}
		case <-authTimer.C:
			log.Info("Refresh authentication tokens timer triggered. Refreshing...")
			err := c.authClient.LoginWithUsernameAndPassword(c.Username, c.Password)
			if err != nil {
				log.Error(err)
			}
			authTimer.Reset(authRefreshTime * time.Minute)
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

	authTimer := time.NewTimer(authRefreshTime * time.Minute)

	for {
		fmt.Printf(
			"%s - queue for battle\n"+
				"%s - auto challenge\n"+
				"%s - challenge specific trainer\n"+
				"%s - auto trade\n"+
				"%s - trade with specific trainer\n"+
				"%s - buy random item\n"+
				"%s - make random microtransaction\n"+
				"%s - try to catch pokemon\n"+
				"%s - raid closest gym\n"+
				"%s - exit\n",
			QueueCmd, ChallengeCmd, ChallengeSpecificTrainerCmd, TradeCmd, TradeSpecificTrainerCmd, StoreCmd, MakeMicrotransactionCmd, CatchCmd, RaidCmd, ExitCmd)

		select {
		case notification := <-c.notificationsChannel:
			c.HandleNotifications(notification, c.operationsChannel, CLI)
		case operation := <-c.operationsChannel:
			exit, err := c.TestOperation(operation)
			if err != nil {
				log.Error(err)
				continue
			} else if exit {
				return
			}
		case <-authTimer.C:
			log.Info("Refresh authentication tokens timer triggered. Refreshing...")
			err := c.authClient.LoginWithUsernameAndPassword(c.Username, c.Password)
			if err != nil {
				log.Error(err)
			}
			authTimer.Reset(authRefreshTime * time.Minute)
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
			log.Error(utils.WrapErrorReadStdin(err))
			return
		}
		trimmed := strings.TrimSpace(command)
		if len(trimmed) > 0 {
			c.operationsChannel <- Operation(trimmed)
		} else {
			c.operationsChannel <- NoOp
		}
	}
}

func (c *NovaPokemonClient) TestOperation(operation Operation) (bool, error) {
	split := strings.Split(string(operation), " ")
	log.Infof("Issued operation: %s, args: %s", split[0], split[1:])
	switch Operation(split[0]) {
	case ChallengeCmd:
		return false, c.startAutoChallenge()
	case ChallengeSpecificTrainerCmd:
		if len(split) > 1 {
			return false, c.ChallengePlayer(split[1])
		} else {
			return false, errors.New("missing trainer name")
		}
	case QueueCmd:
		return false, c.StartAutoBattleQueue()
	case TradeCmd:
		return false, c.startAutoTrade()
	case TradeSpecificTrainerCmd:
		split := strings.Split(string(operation), " ")
		if len(split) > 1 {
			return false, c.StartTradeWithPlayer(split[1])
		} else {
			return false, errors.New("missing trainer name")
		}
	case StoreCmd:
		return false, c.BuyRandomItem()
	case CatchCmd:
		return false, c.CatchWildPokemon()
	case MakeMicrotransactionCmd:
		return false, c.MakeRandomMicrotransaction()
	case RaidCmd:
		return false, c.StartLookForNearbyRaid()
	case NoOp:
		return false, nil
	case ExitCmd:
		return true, nil
	default:
		return false, errorInvalidCommand
	}
}

func (c *NovaPokemonClient) HandleNotifications(notification *utils.Notification, operationsChannel chan Operation,
	clientMode string) {

	var (
		rejected       bool
		rejectedChance float64
	)
	if clientMode == CLI {
		log.Infof("got notification: %s\n"+
			"%s - accept\n"+
			"%s - reject\n", notification.Type, AcceptCmd, RejectCmd)

		select {
		case op := <-operationsChannel:
			switch op {
			case AcceptCmd:
			case RejectCmd:
				rejected = true
			default:
				log.Warnf("invalid notification response: %s", op)
				return
			}
		}
	} else if clientMode == AUTO {
		rejectedChance = rand.Float64()
	}

	switch notification.Type {
	case notifications.WantsToTrade:
		if clientMode == AUTO {
			rejected = rejectedChance < c.config.TradeConfig.AcceptProbability
		}

		err := c.handleTradeNotification(notification, rejected)
		if err != nil {
			log.Error(err)
		}
	case notifications.ChallengeToBattle:
		if clientMode == AUTO {
			rejected = rejectedChance < c.config.BattleConfig.AcceptProbability
		}

		err := c.handleChallengeNotification(notification, rejected)
		if err != nil {
			log.Error(err)
		}
	}
}

func (c *NovaPokemonClient) MakeRandomMicrotransaction() error {
	items, err := c.microtransacitonsClient.GetOffers()
	if err != nil {
		return wrapMakeRandomMicrotransaction(err)
	}

	randomItem := items[rand.Intn(len(items))]
	log.Infof("making purchase of pack %s for %d money, gaining %d coins", randomItem.Name, randomItem.Price, randomItem.Coins)
	transactionId, statsToken, err := c.microtransacitonsClient.PerformTransaction(randomItem.Name, c.authClient.AuthToken, c.trainersClient.TrainerStatsToken)
	if err != nil {
		return wrapMakeRandomMicrotransaction(err)
	}
	log.Infof("Made transaction with id: %s", transactionId.Hex())
	err = c.trainersClient.SetTrainerStatsToken(statsToken)
	if err != nil {
		return wrapMakeRandomMicrotransaction(err)
	}

	return nil
}

func (c *NovaPokemonClient) BuyRandomItem() error {
	items, err := c.storeClient.GetItems(c.authClient.AuthToken)
	if err != nil {
		return wrapBuyRandomItemError(err)
	}

	randomItem := items[rand.Intn(len(items))]
	statsToken, itemsToken, err := c.storeClient.BuyItem(randomItem.Name, c.authClient.AuthToken, c.trainersClient.TrainerStatsToken)
	if err != nil {
		return wrapBuyRandomItemError(err)
	}

	err = c.trainersClient.SetItemsToken(itemsToken)
	if err != nil {
		return wrapBuyRandomItemError(err)
	}

	err = c.trainersClient.SetTrainerStatsToken(statsToken)
	if err != nil {
		return wrapBuyRandomItemError(err)
	}

	return nil
}

func (c *NovaPokemonClient) CatchWildPokemon() error {
	return wrapCatchWildPokemonError(c.locationClient.CatchWildPokemon(c.trainersClient.ItemsToken))
}

func (c *NovaPokemonClient) Finish() {
	log.Warn("Finishing client...")
	close(c.emitFinish)

	<-c.receiveFinish
}

func (c *NovaPokemonClient) startAutoChallenge() error {
	trainers, err := c.notificationsClient.GetOthersListening(c.authClient.AuthToken)
	if err != nil {
		return wrapStartAutoChallengeError(err)
	}

	if len(trainers) == 0 {
		log.Warn("No one to challenge")
		return nil
	} else {
		log.Infof("got %d trainers", len(trainers))
	}

	challengePlayer := trainers[rand.Intn(len(trainers))]
	log.Infof("Challenging %s to battle", challengePlayer)

	err = c.ChallengePlayer(challengePlayer)
	if err != nil {
		return wrapStartAutoChallengeError(err)
	}

	return nil
}

func (c *NovaPokemonClient) ChallengePlayer(otherPlayer string) error {
	pokemonsToUse, pokemonTkns, err := c.getPokemonsForBattle(c.config.BattleConfig.PokemonsPerBattle)

	if err != nil {
		return wrapChallengePlayerError(err)
	}

	conn, channels, requestTimestamp, err := c.battlesClient.ChallengePlayerToBattle(
		c.authClient.AuthToken,
		pokemonTkns,
		c.trainersClient.TrainerStatsToken,
		c.trainersClient.ItemsToken,
		otherPlayer)

	if err != nil {
		return wrapChallengePlayerError(err)
	}

	err = autoManageBattle(c.trainersClient, conn, *channels, pokemonsToUse, requestTimestamp)
	if err != nil {
		return wrapChallengePlayerError(err)
	}

	return nil
}

func (c *NovaPokemonClient) startAutoTrade() error {
	trainers, err := c.notificationsClient.GetOthersListening(c.authClient.AuthToken)
	if err != nil {
		return wrapStartAutoTrade(err)
	}

	if len(trainers) == 0 {
		log.Warn("No one to trade with")
		return nil
	} else {
		log.Infof("got %d trainers", len(trainers))
	}

	tradeWith := trainers[rand.Intn(len(trainers))]
	log.Info("Will trade with ", tradeWith)

	return wrapStartAutoTrade(c.StartTradeWithPlayer(tradeWith))
}

// Notification Handlers

func (c *NovaPokemonClient) handleTradeNotification(notification *utils.Notification, rejected bool) error {
	var content notifications.WantsToTradeContent
	err := json.Unmarshal(notification.Content, &content)
	if err != nil {
		return wrapHandleTradeNotificationError(err)
	}

	lobbyId, err := primitive.ObjectIDFromHex(content.LobbyId)
	if err != nil {
		return wrapHandleTradeNotificationError(err)
	}

	if rejected {
		err = wrapHandleTradeNotificationError(c.RejectTradeWithPlayer(&lobbyId, content.ServerHostname))
	} else {
		err = wrapHandleTradeNotificationError(c.JoinTradeWithPlayer(&lobbyId, content.ServerHostname))
	}

	return err
}

func (c *NovaPokemonClient) handleChallengeNotification(notification *utils.Notification, rejected bool) error {
	log.Info("I was challenged to a battle")

	var content notifications.WantsToBattleContent

	err := json.Unmarshal(notification.Content, &content)
	if err != nil {
		return wrapHandleBattleNotificationError(err)
	}

	pokemonsToUse, pokemonTkns, err := c.getPokemonsForBattle(c.config.BattleConfig.PokemonsPerBattle)
	if err != nil {
		return wrapHandleBattleNotificationError(err)
	}

	if rejected {
		err = c.battlesClient.RejectChallenge(c.authClient.AuthToken, content.LobbyId, content.ServerHostname)
	} else {
		conn, channels, err := c.battlesClient.AcceptChallenge(c.authClient.AuthToken,
			pokemonTkns,
			c.trainersClient.TrainerStatsToken,
			c.trainersClient.ItemsToken,
			content.LobbyId,
			content.ServerHostname)

		if err != nil {
			return wrapHandleBattleNotificationError(err)
		}

		err = autoManageBattle(c.trainersClient, conn, *channels, pokemonsToUse, 0)
	}

	return wrapHandleBattleNotificationError(err)
}

func (c *NovaPokemonClient) StartAutoBattleQueue() error {
	pokemonsToUse, pokemonTkns, err := c.getPokemonsForBattle(c.config.BattleConfig.PokemonsPerBattle)
	if err != nil {
		return wrapStartAutoBattleQueueError(err)
	}

	conn, channels, err := c.battlesClient.QueueForBattle(c.authClient.AuthToken,
		pokemonTkns,
		c.trainersClient.TrainerStatsToken,
		c.trainersClient.ItemsToken)

	if err != nil {
		return wrapStartAutoBattleQueueError(err)
	}

	err = autoManageBattle(c.trainersClient, conn, *channels, pokemonsToUse, 0)
	if err != nil {
		return wrapStartAutoBattleQueueError(err)
	}

	return nil
}

func (c *NovaPokemonClient) StartLookForNearbyRaid() error {
	pokemonsToUse, pokemonTkns, err := c.getPokemonsForBattle(c.config.RaidConfig.PokemonsPerRaid)
	if err != nil {
		return wrapStartLookForRaid(err)
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	gymsWithServer := c.locationClient.Gyms
	for i := 0; i < len(gymsWithServer); i++ {
		gym := gymsWithServer[i].Gym
		serverName := gymsWithServer[i].ServerName
		gymInfo, err := c.gymsClient.GetGymInfo(serverName, gym.Name)
		if err != nil {
			return wrapStartLookForRaid(err)
		}

		if gymInfo.RaidBoss == nil || gymInfo.RaidBoss.HP == 0 {
			log.Info("Raidboss was nil or had no hp")
			continue
		}

		log.Info("ongoing raid :", gymInfo.RaidForming)
		if !gymInfo.RaidForming {
			log.Info("Creating a new raid...")
			if err = c.gymsClient.CreateRaid(serverName, gymInfo.Name); err != nil {
				return wrapStartLookForRaid(err)
			}
		}
		log.Info("Dialing raids...")
		conn, channels, err := c.gymsClient.EnterRaid(c.authClient.AuthToken, pokemonTkns, c.trainersClient.TrainerStatsToken, c.trainersClient.ItemsToken, gym.Name, serverName)
		if err != nil {
			return wrapStartLookForRaid(err)
		}

		err = autoManageBattle(c.trainersClient, conn, *channels, pokemonsToUse, 0)
		if err != nil {
			return wrapStartLookForRaid(err)
		}

		return nil
	}

	log.Warn(wrapStartLookForRaid(errors.New("there are no gyms nearby")))

	return nil
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
		return nil, nil, errorNotEnoughPokemons
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
		return nil, utils.WrapErrorLoadConfigs(err)
	}

	var clientConfig utils.ClientConfig
	err = json.Unmarshal(fileData, &clientConfig)
	if err != nil {
		return nil, utils.WrapErrorLoadConfigs(err)
	}

	log.Infof("Loaded battles client config: %+v", clientConfig.BattleConfig)
	log.Infof("Loaded trades client config: %+v", clientConfig.TradeConfig)
	log.Infof("Loaded gym client config: %+v", clientConfig.RaidConfig)
	log.Infof("Loaded location client config: %+v", clientConfig.LocationConfig)

	return &clientConfig, nil
}
