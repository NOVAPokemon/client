package main

import (
	"flag"
	"fmt"
	"github.com/NOVAPokemon/utils"
	log "github.com/sirupsen/logrus"
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

	parameters := utils.LocationParameters{
		StartingLocation:  utils.Location{},
		MovingSpeed:       0,
		MovingProbability: 0,
	}
	client.StartUpdatingLocation(parameters)

	if auto {
		client.MainLoopAuto()
	} else {
		client.MainLoopCLI()
	}

	client.Finish()
}
