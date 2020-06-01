package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"time"
)

const (
	logsPath = "/logs"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	flag.Usage = func() {
		fmt.Printf("Usage\n")
		fmt.Printf("./client -a \n")
		// flag.PrintDefaults()
	}

	var auto bool
	flag.BoolVar(&auto, "a", false, "start automatic client")
	flag.Parse()

	username := RandomString(20)

	setLogToFile(username)

	client := NovaPokemonClient{
		Username: username,
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

func setLogToFile(username string) {
	filename := fmt.Sprintf("%s/%s.log", logsPath, username)

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(fmt.Sprintf("could not set logger to %s", filename))
	}

	log.SetOutput(file)
}
