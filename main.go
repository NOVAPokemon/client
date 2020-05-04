package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

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
	rand.Seed(time.Now().UnixNano())

	flag.Usage = func() {
		fmt.Printf("Usage\n")
		fmt.Printf("./client -a \n")
		//flag.PrintDefaults()  // prints default usage
	}
	var auto bool
	flag.BoolVar(&auto, "a", false, "start automatic client")
	flag.Parse()

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
	client.StartUpdatingLocation()

	if auto {
		client.MainLoopAuto()
	} else {
		client.MainLoopCLI()
	}

	client.Finish()
}
