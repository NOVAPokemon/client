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
	// storeClient *store.StoreClient // TODO

	jar *cookiejar.Jar
}

func (client *NovaPokemonClient) init() {

	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Error(err)
		return
	}
	client.jar = jar

	client.authClient = &clients.AuthClient{
		Jar: client.jar,
	}

	client.battlesClient = &clients.BattleLobbyClient{
		BattlesAddr: fmt.Sprintf("%s:%d", utils.Host, utils.BattlesPort),
		Jar:         client.jar,
	}

	client.tradesClient = &clients.TradeLobbyClient{
		TradesAddr: fmt.Sprintf("%s:%d", utils.Host, utils.TradesPort),
		Jar:        client.jar,
	}

	client.notificationsClient = &clients.NotificationClient{
		NotificationsAddr: fmt.Sprintf("%s:%d", utils.Host, utils.NotificationsPort),
		Jar:               client.jar,
	}

	client.trainersClient = &clients.TrainersClient{}
	client.trainersClient.Init(fmt.Sprintf("%s:%d", utils.Host, utils.TrainersPort), client.jar)

}

func (client *NovaPokemonClient) StartAutoClient(username string, password string) {
	client.authClient.Register(client.Username, client.Password)
}

func (client *NovaPokemonClient) StartTradeWithPlayer(playerId string) {
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
