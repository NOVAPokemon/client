package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/NOVAPokemon/utils"
	"github.com/NOVAPokemon/utils/clients"
	"github.com/NOVAPokemon/utils/tokens"
	"github.com/NOVAPokemon/utils/websockets"
	"github.com/NOVAPokemon/utils/websockets/battles"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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

func autoManageBattle(trainersClient *clients.TrainersClient, channels clients.BattleChannels, pokemonTkns []string) error {

	pokemons := make(map[string]*utils.Pokemon, len(pokemonTkns))
	for _, tknstr := range pokemonTkns {
		decodedToken, err := tokens.ExtractPokemonToken(tknstr)

		if err != nil {
			log.Error(err)
			return err
		}

		pokemons[decodedToken.Pokemon.Id.Hex()] = &decodedToken.Pokemon
	}

	timer := time.NewTimer(2 * time.Second)
	var started = false
	var selectedPokemon *utils.Pokemon
	var adversaryPokemon *utils.Pokemon

	for ; ; {

		select {

		case <-channels.FinishChannel:
			log.Info("Automatic battle finished")
			return nil


		case msg := <-channels.InChannel:
			msgParsed, err := websockets.ParseMessage(msg)
			if err != nil {
				log.Error(err)
				return err
			}
			switch msgParsed.MsgType {

			case battles.START:
				started = true

			case battles.ERROR:
				switch msgParsed.MsgArgs[0] {

				case battles.ErrPokemonNoHP:
					_, _ = changeActivePokemon(pokemons, channels.OutChannel)
				case battles.ErrPokemonSelectionPhase:
					_, _ = changeActivePokemon(pokemons, channels.OutChannel)
				case battles.ErrNoPokemonSelected:
					selectedPokemon = nil
				case battles.ErrInvalidPokemonSelected:
					selectedPokemon = nil
				default:
					log.Error(msgParsed.MsgArgs[0])
				}

			case battles.STATUS:
				log.Infof(msgParsed.MsgArgs[0])

			case battles.UPDATE_PLAYER_POKEMON:
				pokemon := &utils.Pokemon{}
				err := json.Unmarshal([]byte(strings.TrimSpace(msgParsed.MsgArgs[0])), pokemon)
				if err != nil {
					log.Error("Error decoding player pokemon")
					return err
				}
				selectedPokemon = pokemon

			case battles.UPDATE_ADVERSARY_POKEMON:
				pokemon := &utils.Pokemon{}
				err := json.Unmarshal([]byte(strings.TrimSpace(msgParsed.MsgArgs[0])), pokemon)
				if err != nil {
					log.Error("Error decoding adversary pokemon")
					return err
				}
				adversaryPokemon = pokemon
				log.Infof("Adversary pokemon:\tID:%s, HP: %d, Species: %s", adversaryPokemon.Id.Hex(),
					adversaryPokemon.HP,
					adversaryPokemon.Species)

			case battles.SET_TOKEN:
				pokemonTkns = msgParsed.MsgArgs
				for _, tkn := range pokemonTkns {

					if len(tkn) == 0 {
						continue
					}

					log.Info(tkn)
					decodedToken, err := tokens.ExtractPokemonToken(tkn)
					if err != nil {
						log.Error(err)
						continue
					}
					trainersClient.PokemonClaims[decodedToken.Pokemon.Id.Hex()] = decodedToken
					trainersClient.PokemonTokens[decodedToken.Pokemon.Id.Hex()] = tkn
				}

				log.Warn("Updated Token!")

			case battles.FINISH:
				log.Warn("Battle finished!")
				return nil
			}

		case <-timer.C:
			// if the battle hasnt started but the pokemon is already picked, do nothing
			log.Info(started, selectedPokemon)
			if started || selectedPokemon == nil {
				err := doNextBattleMove(selectedPokemon, pokemons, channels.OutChannel)
				if err != nil {
					log.Error(err)
					return err
				}
			} else {
				log.Info("Waiting on other player")
			}

			cooldownDuration := time.Duration(RandInt(1500, 2500))
			timer.Reset(cooldownDuration * time.Millisecond)
		}
	}
}

func doNextBattleMove(selectedPokemon *utils.Pokemon, pokemons map[string]*utils.Pokemon, outChannel chan *string) error {

	if selectedPokemon == nil || selectedPokemon.HP == 0 {
		newPokemon, err := changeActivePokemon(pokemons, outChannel)

		if err != nil {
			return err
		}

		log.Infof("Selected pokemon: %s , HP:%d, Damage:%d, Species:%s ",
			newPokemon.Id.Hex(),
			newPokemon.HP,
			newPokemon.Damage,
			newPokemon.Species)

		return nil
	}

	log.Info("Attacking...")
	toSend := websockets.Message{MsgType: battles.ATTACK, MsgArgs: []string{}}
	websockets.SendMessage(toSend, outChannel)

	return nil
}

func changeActivePokemon(pokemons map[string]*utils.Pokemon, outChannel chan *string) (*utils.Pokemon, error) {

	nextPokemon, err := getAlivePokemon(pokemons)
	if err != nil {
		log.Error("No pokemons alive")
		return nil, err
	}
	toSend := websockets.Message{MsgType: battles.SELECT_POKEMON, MsgArgs: []string{nextPokemon.Id.Hex()}}
	websockets.SendMessage(toSend, outChannel)
	return nextPokemon, nil
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

func RandInt(min int, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return min + rand.Intn(max-min)
}
