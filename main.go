package main

import (
	log "github.com/sirupsen/logrus"
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

	client.StartListeningToNotifications()
	client.ParseReceivedNotifications()
	client.Finish()
}
