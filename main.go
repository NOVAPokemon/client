package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	logsPath = "/logs"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	flag.Usage = func() {
		fmt.Printf("Usage\n")
		fmt.Printf("./client -a -l \n")
		// flag.PrintDefaults()
	}

	var auto bool
	flag.BoolVar(&auto, "a", false, "start automatic client")

	var logToStdout bool
	flag.BoolVar(&logToStdout, "l", false, "log to stdout")

	var clientNum int
	flag.IntVar(&clientNum, "n", -1, "client thread number")

	flag.Parse()

	username := RandomString(20)

	if !logToStdout {
		setLogToFile(username)
	}

	if clientNum != -1 {
		log.Infof("Thread number: %d", clientNum)
	}

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

	file, err := os.Create(filename)
	if err != nil {
		panic(fmt.Sprintf("could not set logger to %s on creation", filename))
	}

	err = file.Chmod(0666)
	if err != nil {
		panic(fmt.Sprintf("could not set logger to %s due to chmod changes", filename))
	}

	log.SetOutput(file)
}
