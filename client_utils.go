package main

import (
	"bufio"
	"fmt"
	"github.com/NOVAPokemon/utils/clients"
	"github.com/NOVAPokemon/utils/notifications"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"math/rand"
	"os"
	"strings"
	"time"
)

func requestUsername() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter username: ")
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func requestPassword() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter password: ")
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func waitForBattleChallenges(c *NovaPokemonClient) error {

	for ; ; {
		notification := <-c.notificationsClient.NotificationsChannel
		switch notification.Type {
		case notifications.ChallengeToBattle:
			log.Info("I was challenged to a battle")
			battleId, err := primitive.ObjectIDFromHex(string(notification.Content))
			if err != nil {
				log.Error(err)
				return err
			}
			c.battlesClient.AcceptChallenge(battleId)
		}
	}

}

func autoManageBattle(c *NovaPokemonClient, channels clients.BattleChannels) error {

	go func() {

	}()

	for ; ; {
		select {

		case <-channels.FinishChannel:


		case <-channels.Channel:



		}

	}
}

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	rand.Seed(time.Now().Unix())

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
