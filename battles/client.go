package battles

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/NOVAPokemon/utils"
	"github.com/NOVAPokemon/utils/websockets/battles"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"
)

type BattleLobbyClient struct {
	BattlesAddr string
	Jar         *cookiejar.Jar
	conn        *websocket.Conn
}

func (client *BattleLobbyClient) GetAvailableLobbies() []utils.Lobby {

	u := url.URL{Scheme: "http", Host: client.BattlesAddr, Path: "/battles"}

	resp, err := http.Get(u.String())

	if err != nil {
		log.Error(err)
		return nil
	}

	var availableBattles []utils.Lobby
	err = json.NewDecoder(resp.Body).Decode(&availableBattles)

	if err != nil {
		log.Error(err)
		return nil
	}

	return availableBattles
}

func (client *BattleLobbyClient) CreateBattleLobby() {

	u := url.URL{Scheme: "ws", Host: client.BattlesAddr, Path: "/battles/join"}
	log.Infof("Connecting to: %s", u.String())

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		Jar:              client.Jar,
	}

	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()
	client.conn = c

	defer c.Close()
	client.conn = c

	inChannel := make(chan *string)
	finished := make(chan bool)
	go handleRecv(c, inChannel)
	go readMessages(inChannel, finished)

	mainLoop(c, finished)

}

func (client *BattleLobbyClient) JoinBattleLobby(battleId primitive.ObjectID) {

	u := url.URL{Scheme: "ws", Host: client.BattlesAddr, Path: "/battles/join/" + battleId.Hex()}
	log.Infof("Connecting to: %s", u.String())

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		Jar:              client.Jar,
	}

	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()
	client.conn = c

	defer c.Close()
	client.conn = c

	inChannel := make(chan *string)
	finished := make(chan bool)

	go handleRecv(c, inChannel)
	go readMessages(inChannel, finished)

	mainLoop(c, finished)

}

// adapted from trades client

func send(conn *websocket.Conn, msg *string) {
	err := conn.WriteMessage(websocket.TextMessage, []byte(*msg))

	if err != nil {
		return
	} else {
		log.Debugf("Wrote %s into the channel", *msg)
	}
}

func handleRecv(conn *websocket.Conn, channel chan *string) {
	defer close(channel)

	for {
		_, message, err := conn.ReadMessage()

		if err != nil {
			return
		} else {
			msg := string(message)
			log.Debugf("Received %s from the websocket", msg)
			channel <- &msg
		}
	}
}

func readMessages(inChannel chan *string, finished chan bool) {
	defer close(finished)

	for {
		select {
		case msg, ok := <-inChannel:
			if !ok {
				return
			}

			err, battleMsg := battles.ParseMessage(msg)
			if err != nil {
				log.Error(err)
				continue
			}

			log.Infof("Message: %s", *msg)

			if battleMsg.MsgType == battles.ERROR { //TODO
				log.Info("Error")
				finished <- true
				return
			}
		}
	}
}

func finish(conn *websocket.Conn) {
	err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close:", err)
		return
	}
}

func mainLoop(conn *websocket.Conn, finished chan bool) {
	for {
		select {
		case v := <-finished:
			if v == true {
				finish(conn)
				return
			}
		default:
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter text: ")
			text, _ := reader.ReadString('\n')
			send(conn, &text)
		}
	}
}
