package main

import (
	"bufio"
	"fmt"
	"github.com/NOVAPokemon/utils/clients"
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

func GetRandomElementsFromArray(arr []interface{}, count int) []interface{} {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(arr), func(i, j int) { arr[i], arr[j] = arr[j], arr[i] })
	return arr[:count]
}
