package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/NOVAPokemon/utils/clients"
	"github.com/NOVAPokemon/utils/pokemons"
	"github.com/NOVAPokemon/utils/tokens"
	"github.com/NOVAPokemon/utils/websockets"
	"github.com/NOVAPokemon/utils/websockets/battles"
	"github.com/gorilla/websocket"
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

func autoManageBattle(trainersClient *clients.TrainersClient, conn *websocket.Conn, channels clients.BattleChannels, pokemonTkns []string) error {
	defer conn.Close()

	pokemonsMap := make(map[string]*pokemons.Pokemon, len(pokemonTkns))
	for _, tknstr := range pokemonTkns {
		decodedToken, err := tokens.ExtractPokemonToken(tknstr)

		if err != nil {
			log.Error(err)
			return err
		}
		pokemonsMap[decodedToken.Pokemon.Id.Hex()] = &decodedToken.Pokemon
	}

	const timeout = 10 * time.Second

	cdTimer := time.NewTimer(2 * time.Second)
	expireTimer := time.NewTimer(timeout)

	var startTime = time.Now()
	var started = false
	var selectedPokemon *pokemons.Pokemon
	var adversaryPokemon *pokemons.Pokemon

	go func() {
		<-expireTimer.C
		if !started {
			log.Warn("Leaving lobby because other player hasn't joined")
			conn.Close()
		}
	}()

	for {
		select {

		case <-channels.FinishChannel:
			return nil

		case <-cdTimer.C:
			// if the battle hasn't started but the pokemon is already picked, do nothing
			if started {
				err := doNextBattleMove(selectedPokemon, pokemonsMap, channels.OutChannel)
				if err != nil {
					log.Error(err)
					close(channels.FinishChannel)
					conn.Close()
					return err
				}
			} else {
				remainingTime := time.Until(startTime.Add(timeout))
				log.Infof("Waiting on other player, timing out in %f seconds", remainingTime.Seconds())
			}

			cooldownDuration := time.Duration(RandInt(350, 750))
			cdTimer.Reset(cooldownDuration * time.Millisecond)

		case msg, ok := <-channels.InChannel:

			if !ok {
				continue
			}

			msgParsed, err := websockets.ParseMessage(msg)

			if err != nil {
				return err
			}

			switch msgParsed.MsgType {
			case battles.START:
				started = true

			case battles.ERROR:
				switch msgParsed.MsgArgs[0] {
				case battles.ErrNoPokemonSelected:
					selectedPokemon = nil
				case battles.ErrInvalidPokemonSelected:
					selectedPokemon = nil
				default:
					log.Warn(msgParsed.MsgArgs[0])
				}

			case battles.UPDATE_PLAYER_POKEMON:
				pokemon := &pokemons.Pokemon{}
				err := json.Unmarshal([]byte(strings.TrimSpace(msgParsed.MsgArgs[0])), pokemon)
				if err != nil {
					log.Error("Error decoding player pokemon")
					<-channels.FinishChannel
					return err
				}
				log.Infof("Self pokemon:\tID:%s, HP: %d, Species: %s", pokemon.Id.Hex(),
					pokemon.HP,
					pokemon.Species)

				selectedPokemon = pokemon
				pokemonsMap[pokemon.Id.Hex()] = pokemon

			case battles.UPDATE_ADVERSARY_POKEMON:
				pokemon := &pokemons.Pokemon{}
				err := json.Unmarshal([]byte(strings.TrimSpace(msgParsed.MsgArgs[0])), pokemon)
				if err != nil {
					log.Error("Error decoding adversary pokemon")
					<-channels.FinishChannel
					return err
				}
				adversaryPokemon = pokemon
				log.Infof("Adversary pokemon:\tID:%s, HP: %d, Species: %s", adversaryPokemon.Id.Hex(),
					adversaryPokemon.HP,
					adversaryPokemon.Species)

			case battles.SET_TOKEN:
				tknType := msgParsed.MsgArgs[0]

				switch tknType {
				case tokens.StatsTokenHeaderName:
					decodedToken, err := tokens.ExtractStatsToken(msgParsed.MsgArgs[1])
					if err != nil {
						log.Error(err)
						continue
					}
					trainersClient.TrainerStatsClaims = decodedToken
					trainersClient.TrainerStatsToken = msgParsed.MsgArgs[1]

				case tokens.PokemonsTokenHeaderName:
					pokemonTkns = msgParsed.MsgArgs[1:]
					for _, tkn := range pokemonTkns {

						if len(tkn) == 0 {
							continue
						}
						decodedToken, err := tokens.ExtractPokemonToken(tkn)
						if err != nil {
							log.Error(err)
							continue
						}
						trainersClient.PokemonClaims[decodedToken.Pokemon.Id.Hex()] = decodedToken
						trainersClient.PokemonTokens[decodedToken.Pokemon.Id.Hex()] = tkn
					}

				}
				log.Warn("Updated Token!")
			}
		}
	}
}

func doNextBattleMove(selectedPokemon *pokemons.Pokemon, pokemons map[string]*pokemons.Pokemon, outChannel chan *string) error {
	if selectedPokemon == nil || selectedPokemon.HP == 0 {
		err := changeActivePokemon(pokemons, outChannel)
		if err != nil {
			return err
		}
		return nil
	}
	log.Info("Attacking...")
	toSend := websockets.Message{MsgType: battles.ATTACK, MsgArgs: []string{}}
	websockets.SendMessage(toSend, outChannel)
	return nil
}

func changeActivePokemon(pokemons map[string]*pokemons.Pokemon, outChannel chan *string) error {
	nextPokemon, err := getAlivePokemon(pokemons)
	if err != nil {
		log.Error("No pokemons alive")
		return err
	}
	log.Infof("Selecting pokemon:\tID:%s, HP: %d, Species: %s", nextPokemon.Id.Hex(),
		nextPokemon.HP,
		nextPokemon.Species)
	toSend := websockets.Message{MsgType: battles.SELECT_POKEMON, MsgArgs: []string{nextPokemon.Id.Hex()}}
	websockets.SendMessage(toSend, outChannel)
	return nil
}

func getAlivePokemon(pokemons map[string]*pokemons.Pokemon) (*pokemons.Pokemon, error) {

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
