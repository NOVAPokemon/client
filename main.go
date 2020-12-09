package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/NOVAPokemon/utils"
	"github.com/NOVAPokemon/utils/websockets"
	log "github.com/sirupsen/logrus"
)

const (
	logsPath                       = "/logs"
	defaultLocationWeightsFilename = "location_weights.json"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	flag.Usage = func() {
		fmt.Printf("Usage\n")
		fmt.Println("./client -a -l")
		// flag.PrintDefaults()
	}

	var (
		autoClient   bool
		logToStdout  bool
		clientNum    int
		regionTag    string
		commsManager websockets.CommunicationManager
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

	if regionTag == "" {
		log.Info("starting client without any region associated")
		commsManager = utils.CreateDefaultCommunicationManager()
	} else {
		commsManager = utils.CreateDefaultDelayedManager(true)
	}

	client := novaPokemonClient{
		Username: username,
		Password: randomString(20),
	}

	client.init(commsManager, regionTag)

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

func getRandomRegion(locationWeights utils.LocationWeights) string {
	encodedRegions := map[int]string{}
	var encodedRegionsMultByWeight []int
	encodedRegionsMultByWeight = []int{}
	encodedValue := 0

	log.Info("location weights: ", locationWeights)

	for regionName, weight := range locationWeights {
		encodedRegions[encodedValue] = regionName
		for i := 0; i < weight; i++ {
			encodedRegionsMultByWeight = append(encodedRegionsMultByWeight, encodedValue)
		}
		encodedValue++
	}

	randIdx := rand.Intn(len(encodedRegionsMultByWeight))
	randRegionEncoded := encodedRegionsMultByWeight[randIdx]

	randRegion, ok := encodedRegions[randRegionEncoded]
	if !ok {
		panic(fmt.Sprintf("no region matched encoded %d", randRegionEncoded))
	}
	return randRegion
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

func loadLocationWeights(locationWeightsFilename string) utils.LocationWeights {
	fileData, err := ioutil.ReadFile(locationWeightsFilename)
	if err != nil {
		log.Error("error loading regions filename")
		panic(err)
	}

	var locationWeights utils.LocationWeights
	err = json.Unmarshal(fileData, &locationWeights)
	if err != nil {
		panic(err)
		return nil
	}

	return locationWeights
}
