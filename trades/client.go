package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/NOVAPokemon/utils"
	"github.com/NOVAPokemon/utils/websockets/trades"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"
)

type TradeLobbyClient struct {
	HubAddr string

	TradeId primitive.ObjectID

	Self     utils.Trainer
	Trainer2 utils.Trainer

	started  bool
	finished bool

	conn *websocket.Conn

	jar *cookiejar.Jar
}

func NewTradeLobbyClient(hubAddr string, self utils.Trainer, jar *cookiejar.Jar) *TradeLobbyClient {
	return &TradeLobbyClient{
		HubAddr:  hubAddr,
		Self:     self,
		started:  false,
		finished: false,
		jar:      jar,
	}
}

func GetAvailableLobbies(client *TradeLobbyClient) []utils.Lobby {

	u := url.URL{Scheme: "http", Host: client.HubAddr, Path: "/trades"}

	httpClient := &http.Client{
		Jar: client.jar,
	}

	resp, err := httpClient.Get(u.String())

	if err != nil {
		log.Error(err)
		return nil
	}

	var battles []utils.Lobby
	err = json.NewDecoder(resp.Body).Decode(&battles)

	if err != nil {
		log.Error(err)
		return nil
	}

	return battles
}

func CreateTradeLobby(client *TradeLobbyClient) {
	u := url.URL{Scheme: "ws", Host: client.HubAddr, Path: "/trades/join"}
	log.Infof("Connecting to: %s", u.String())

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		Jar:              client.jar,
	}

	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()
	client.conn = c

	finished := make(chan struct{})
	writeChannel := make(chan *string)

	go readMessages(c, finished)
	go writeMessage(writeChannel)

	mainLoop(c, writeChannel, finished)

	log.Info("Finishing...")
}

func JoinTradeLobby(client *TradeLobbyClient, battleId primitive.ObjectID) {
	u := url.URL{Scheme: "ws", Host: client.HubAddr, Path: "/trades/join/" + battleId.Hex()}
	log.Infof("Connecting to: %s", u.String())

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		Jar:              client.jar,
	}

	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()
	client.conn = c

	finished := make(chan struct{})
	writeChannel := make(chan *string)

	go readMessages(c, finished)
	go writeMessage(writeChannel)

	mainLoop(c, writeChannel, finished)

	log.Info("Finishing...")
}

func send(conn *websocket.Conn, msg *string) {
	err := conn.WriteMessage(websocket.TextMessage, []byte(*msg))

	if err != nil {
		return
	} else {
		log.Debugf("Wrote %s into the channel", msg)
	}
}

func readMessages(conn *websocket.Conn, finished chan struct{}) {
	defer close(finished)

	for {
		_, message, err := conn.ReadMessage()

		if err != nil {
			log.Error(err)
			return
		}

		msg := string(message)
		log.Debugf("Received %s from the websocket", msg)

		err, tradeMsg := trades.ParseMessage(&msg)
		if err != nil {
			log.Error(err)
			continue
		}

		log.Infof("Message: %s", msg)

		if tradeMsg.MsgType == trades.FINISH {
			log.Info("Finished trade.")
			return
		}
	}
}

func writeMessage(writeChannel chan *string) {
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		text, _ := reader.ReadString('\n')
		writeChannel <- &text
	}
}

func finish() {
	log.Info("Finishing connection...")
}

func mainLoop(conn *websocket.Conn, writeChannel chan *string, finished chan struct{}) {
	for {
		select {
		case <-finished:
			finish()
			return
		case msg := <-writeChannel:
			send(conn, msg)
		}
	}
}
