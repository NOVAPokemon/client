package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/NOVAPokemon/utils"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"net/url"
	"os"
)

type BattleLobbyClient struct {
	HubAddr string

	BattleId primitive.ObjectID

	Self     utils.Trainer
	Trainer2 utils.Trainer

	started  bool
	finished bool

	conn *websocket.Conn
}

func NewBattleLobbyClient(hubAddr string, self utils.Trainer) *BattleLobbyClient {
	return &BattleLobbyClient{
		HubAddr:  hubAddr,
		Self:     self,
		started:  false,
		finished: false,
	}
}

func GetAvailableLobbies(client *BattleLobbyClient) []utils.Lobby {

	u := url.URL{Scheme: "http", Host: client.HubAddr, Path: "/battles"}

	resp, err := http.Get(u.String())

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

func CreateBattleLobby(client *BattleLobbyClient) {

	u := url.URL{Scheme: "ws", Host: client.HubAddr, Path: "/battles/join"}
	log.Infof("Connecting to: %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()
	client.conn = c

	inChannel := make(chan *string)
	outChannel := make(chan *string)
	go handleRecv(c, inChannel)
	go handleSend(c, outChannel)

	go func() {
		for {
			select {
			case msg := <-inChannel:
				log.Infof("Message from trainer 1 received: %s", *msg)
			}
		}
	}()

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		text, _ := reader.ReadString('\n')
		outChannel <- &text
	}

}

func JoinBattleLobby(client *BattleLobbyClient, battleId primitive.ObjectID) {

	u := url.URL{Scheme: "ws", Host: client.HubAddr, Path: "/battles/join/" + battleId.Hex()}
	log.Infof("Connecting to: %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()
	client.conn = c

	inChannel := make(chan *string)
	outChannel := make(chan *string)
	go handleRecv(c, inChannel)
	go handleSend(c, outChannel)

	go func() {
		for {
			select {
			case msg := <-inChannel:
				log.Infof("Message received: %s", *msg)
			}
		}
	}()

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		text, _ := reader.ReadString('\n')
		outChannel <- &text
	}

}

func handleSend(conn *websocket.Conn, channel chan *string) {

	for {
		msg := <-channel

		err := conn.WriteMessage(websocket.TextMessage, []byte(*msg))

		if err != nil {
			log.Error("write err:", err)
		} else {
			log.Debugf("Wrote %s into the channel", msg)
		}

	}

}

func handleRecv(conn *websocket.Conn, channel chan *string) {

	for {
		_, message, err := conn.ReadMessage()

		if err != nil {
			log.Println(err)
		} else {
			msg := string(message)
			log.Debugf("Received %s from the websocket", msg)
			channel <- &msg

		}

	}

}
