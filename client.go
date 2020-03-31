package main

import (
	"fmt"
	"github.com/NOVAPokemon/client/auth"
	"github.com/NOVAPokemon/client/battles"
	"github.com/NOVAPokemon/client/notifications"
	"github.com/NOVAPokemon/client/trades"
	"github.com/NOVAPokemon/client/trainers"
	"github.com/NOVAPokemon/utils"
	"net/http/cookiejar"
)

type NovaPokemonClient struct {
	Username string
	Password string

	authClient          *auth.Client
	battlesClient       *battles.BattleLobbyClient
	tradesClient        *trades.TradeLobbyClient
	notificationsClient *notifications.NotificationClient
	trainersClient      *trainers.TrainersClient
	jar                 *cookiejar.Jar
	// storeClient *store.StoreClient // TODO
}

func (client *NovaPokemonClient) init() {

	client.jar, _ = cookiejar.New(nil)

	client.authClient = &auth.Client{
		Jar:         client.jar,
	}

	client.battlesClient = &battles.BattleLobbyClient{
		BattlesAddr: fmt.Sprintf("%s:%d", utils.Host, utils.BattlesPort),
		Jar:         client.jar,
	}

	client.tradesClient = &trades.TradeLobbyClient{
		TradesAddr: fmt.Sprintf("%s:%d", utils.Host, utils.TradesPort),
		Jar:        client.jar,
	}

	client.notificationsClient = &notifications.NotificationClient{
		NotificationsAddr: fmt.Sprintf("%s:%d", utils.Host, utils.NotificationsPort),
		Jar:               client.jar,
	}

	client.trainersClient = &trainers.TrainersClient{
		TrainersAddr: fmt.Sprintf("%s:%d", utils.Host, utils.TrainersPort),
	}

}

func (client *NovaPokemonClient) Register(username string, password string) {
	//client.authClient.Register(client.Username, client.Password)
}

func (client *NovaPokemonClient) Login() {
	client.authClient.LoginWithUsernameAndPassword(client.Username, client.Password)
}

func (client *NovaPokemonClient) GetAllTokens() error {
	return client.authClient.GetInitialTokens(client.Username)
}

