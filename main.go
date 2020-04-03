package main

import (
	log "github.com/sirupsen/logrus"
	"net/url"
)

//
//func main() {
//
//	client := NovaPokemonClient{
//		Username: requestUsername(),
//		Password: requestPassword(),
//	}
//
//	client.init()
//	_ = LoginAndStartAutoBattleQueue(&client)
//
//}

func main() {
	client := NovaPokemonClient{
		Username: RandomString(20),
		Password: RandomString(20),
	}
	client.init()

	err := client.RegisterAndGetTokens()
	if err != nil {
		log.Error(err)
		return
	}

	err = client.LoginAndGetTokens()
	if err != nil {
		log.Error(err)
		return
	}

	for _, cookie := range client.jar.Cookies(&url.URL{
		Scheme: "http",
		Host:   "localhost",
	}) {
		log.Info(cookie.Name)
	}

	client.StartListeningToNotifications()
	client.ParseReceivedNotifications()
}
