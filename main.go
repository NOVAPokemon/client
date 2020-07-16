package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/NOVAPokemon/utils"
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

	var (
		autoClient   bool
		logToStdout  bool
		clientNum    int
		regionTag    string
		commsManager utils.CommunicationManager
	)

	flag.BoolVar(&autoClient, "a", false, "start automatic client")
	flag.BoolVar(&logToStdout, "l", false, "log to stdout")
	flag.IntVar(&clientNum, "n", -1, "client thread number")
	flag.StringVar(&regionTag, "r", "", "region tag for client")

	flag.Parse()

	username := randomString(20)

	if !logToStdout {
		setLogToFile(username)
	}

	if clientNum != -1 {
		log.Infof("Thread number: %d", clientNum)
	}

	if regionTag != "" {
		log.Infof("starting client in region %s", regionTag)
		commsManager = utils.CreateDelayedCommunicationManager(utils.DefaultDelayConfigFilename, regionTag)
	} else {
		log.Info("starting client without any region associated")
		commsManager = utils.CreateDefaultCommunicationManager()
	}

	client := novaPokemonClient{
		Username: username,
		Password: randomString(20),
	}
	client.init(commsManager)

	err := client.registerAndGetTokens()
	if err != nil {
		log.Error(err)
		return
	}

	client.startListeningToNotifications()
	client.startUpdatingLocation()

	if autoClient {
		client.mainLoopAuto()
	} else {
		client.mainLoopCLI()
	}

	client.finish()
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
