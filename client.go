package main

import (
	"fmt"
	"github.com/NOVAPokemon/utils"
	"github.com/NOVAPokemon/utils/clients"
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

func (client *NovaPokemonClient) init() {

	client.jar, _ = cookiejar.New(nil)

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

	client.trainersClient = &clients.TrainersClient{
		TrainersAddr: fmt.Sprintf("%s:%d", utils.Host, utils.TrainersPort),
	}

}

func (c *NovaPokemonClient) StartAutoClient(username string, password string) {
	c.authClient.Register(c.Username, c.Password)
}

func (c *NovaPokemonClient) StartTradeWithPlayer(playerId string) {
}

func (c *NovaPokemonClient) Register() {
	c.authClient.Register(c.Username, c.Password)
}

func (c *NovaPokemonClient) Login() {
	c.authClient.LoginWithUsernameAndPassword(c.Username, c.Password)
}

func (c *NovaPokemonClient) GetAllTokens() error {
	return c.authClient.GetInitialTokens(c.Username)
}
