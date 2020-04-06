package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/NOVAPokemon/utils"
	"github.com/NOVAPokemon/utils/clients"
	"github.com/NOVAPokemon/utils/websockets"
	"github.com/NOVAPokemon/utils/websockets/battles"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

func autoManageBattle(channels clients.BattleChannels, pokemons map[string]*utils.Pokemon) error {

	timer := time.NewTimer(2 * time.Second)

	selectedPokemon, err := getAlivePokemon(pokemons)

	if err != nil {
		logrus.Error("No pokemons alive")
		return err
	}

	logrus.Infof("Sent selection message")
	toSend := websockets.Message{MsgType: battles.SELECT_POKEMON, MsgArgs: []string{selectedPokemon.Id.Hex()}}
	websockets.SendMessage(toSend, channels.OutChannel)
	timer.Reset(2 * time.Second)

	for ; ; {

		select {

		case <-channels.FinishChannel:
			logrus.Info("Automatic battle finished")
			return nil

		case <-timer.C:
			logrus.Info("Attacking...")
			toSend := websockets.Message{MsgType: battles.ATTACK, MsgArgs: []string{}}
			websockets.SendMessage(toSend, channels.OutChannel)
			timer.Reset(2 * time.Second)


		case msg := <-channels.InChannel:
			msgParsed, err := websockets.ParseMessage(msg)

			if err != nil {
				logrus.Error(err)
				return err
			}

			switch msgParsed.MsgType {

			case battles.ERROR:
				switch msgParsed.MsgArgs[0] {

				case battles.ErrPokemonNoHP:
					selectedPokemon, err = getAlivePokemon(pokemons)
					if err != nil {
						logrus.Error("No pokemons alive")
						return err
					}
					toSend := websockets.Message{MsgType: battles.SELECT_POKEMON, MsgArgs: []string{selectedPokemon.Id.Hex()}}
					websockets.SendMessage(toSend, channels.OutChannel)

				case battles.ErrPokemonSelectionPhase:
					selectedPokemon, err = getAlivePokemon(pokemons)
					if err != nil {
						logrus.Error("No pokemons alive")
						return err
					}
					toSend := websockets.Message{MsgType: battles.SELECT_POKEMON, MsgArgs: []string{selectedPokemon.Id.Hex()}}
					websockets.SendMessage(toSend, channels.OutChannel)

				case battles.ErrNoPokemonSelected:
					selectedPokemon, err = getAlivePokemon(pokemons)
					if err != nil {
						logrus.Error("No pokemons alive")
						return err
					}
					toSend := websockets.Message{MsgType: battles.SELECT_POKEMON, MsgArgs: []string{selectedPokemon.Id.Hex()}}
					websockets.SendMessage(toSend, channels.OutChannel)

				case battles.ErrInvalidPokemonSelected:
					selectedPokemon, err = getAlivePokemon(pokemons)
					if err != nil {
						logrus.Error("No pokemons alive")
						return err
					}
					toSend := websockets.Message{MsgType: battles.SELECT_POKEMON, MsgArgs: []string{selectedPokemon.Id.Hex()}}
					websockets.SendMessage(toSend, channels.OutChannel)

				default:
					logrus.Error(msgParsed.MsgArgs[0])
				}
			case battles.STATUS:

			case battles.UPDATE_PLAYER:

			case battles.UPDATE_ADVERSARY:

			case battles.SELECT_POKEMON:

			case battles.UPDATE_ADVERSARY_POKEMON:

			case battles.UPDATE_PLAYER_POKEMON:
				pokemon := &utils.Pokemon{}
				fmt.Println(msgParsed)
				logrus.Infof("Decoding : %+v", msgParsed.MsgArgs[0])

				err := json.Unmarshal([]byte(strings.TrimSpace(msgParsed.MsgArgs[0])), pokemon)

				if err != nil {
					logrus.Error("Error decoding pokemon")
					return err
				}

				selectedPokemon = pokemon
			case battles.FINISH:
				return nil
			}
		}
	}

}

func getAlivePokemon(pokemons map[string]*utils.Pokemon) (*utils.Pokemon, error) {

	for _, v := range pokemons {
		if v.HP > 0 {
			return v, nil
		}
	}

	return nil, errors.New("No pokemons alive")
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
