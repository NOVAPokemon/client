package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/NOVAPokemon/utils/clients"
	"github.com/NOVAPokemon/utils/items"
	"github.com/NOVAPokemon/utils/pokemons"
	"github.com/NOVAPokemon/utils/tokens"
	"github.com/NOVAPokemon/utils/websockets"
	"github.com/NOVAPokemon/utils/websockets/battles"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"math"
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

func autoManageBattle(trainersClient *clients.TrainersClient, conn *websocket.Conn, channels clients.BattleChannels, chosenPokemons map[string]*pokemons.Pokemon) error {
	defer conn.Close()
	rand.Seed(time.Now().Unix())
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
				err := doNextBattleMove(selectedPokemon, chosenPokemons, trainersClient.ItemsClaims.Items, channels.OutChannel)
				if err != nil {
					log.Error(err)
					continue
				}
			} else {
				remainingTime := time.Until(startTime.Add(timeout))
				log.Infof("Waiting on other player, timing out in %f seconds", remainingTime.Seconds())
			}

			cooldownDuration := time.Duration(RandInt(1000, 1500))
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
				log.Infof("Self pokemon:\tID:%s, HP: %d, maxHP :%d Species: %s", pokemon.Id.Hex(),
					pokemon.HP,
					pokemon.MaxHP,
					pokemon.Species)

				chosenPokemons[pokemon.Id.Hex()] = pokemon
				selectedPokemon = pokemon

			case battles.UPDATE_ADVERSARY_POKEMON:
				pokemon := &pokemons.Pokemon{}
				err := json.Unmarshal([]byte(strings.TrimSpace(msgParsed.MsgArgs[0])), pokemon)
				if err != nil {
					log.Error("Error decoding adversary pokemon")
					<-channels.FinishChannel
					return err
				}
				adversaryPokemon = pokemon
				log.Infof("Adversary pokemon:\tID:%s, HP: %d, maxHP :%d Species: %s", adversaryPokemon.Id.Hex(),
					adversaryPokemon.HP,
					adversaryPokemon.MaxHP,
					adversaryPokemon.Species)

			case battles.REMOVE_ITEM:
				if len(msgParsed.MsgArgs) > 0 {
					delete(trainersClient.ItemsClaims.Items, msgParsed.MsgArgs[0])
				}
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
					pokemonTkns := msgParsed.MsgArgs[1:]
					for _, tkn := range pokemonTkns {

						if len(tkn) == 0 {
							continue
						}
						decodedToken, err := tokens.ExtractPokemonToken(tkn)
						if err != nil {
							log.Error(err)
							continue
						}
						trainersClient.PokemonClaims[decodedToken.Pokemon.Id.Hex()] = *decodedToken
						trainersClient.PokemonTokens[decodedToken.Pokemon.Id.Hex()] = tkn

					}
				case tokens.ItemsTokenHeaderName:
					decodedToken, err := tokens.ExtractItemsToken(msgParsed.MsgArgs[1])
					if err != nil {
						log.Error(err)
						continue
					}
					trainersClient.ItemsClaims = decodedToken
					trainersClient.ItemsToken = msgParsed.MsgArgs[1]

				}
				log.Warn("Updated Token!")
			}
		}
	}
}

func doNextBattleMove(selectedPokemon *pokemons.Pokemon, trainerPokemons map[string]*pokemons.Pokemon, trainerItems map[string]items.Item, outChannel chan *string) error {
	if selectedPokemon == nil {
		newPokemon, err := changeActivePokemon(trainerPokemons, outChannel)
		if err != nil {
			return err
		}
		selectedPokemon = newPokemon
		return nil
	}

	if selectedPokemon.HP == 0 {
		// see if we have revive
		itemToUse, err := getReviveItem(trainerItems)
		if err == nil {
			log.Info("no revive items left")

			log.Info("Using revive...")
			toSend := websockets.Message{MsgType: battles.USE_ITEM, MsgArgs: []string{itemToUse.Id.Hex()}}
			websockets.SendMessage(toSend, outChannel)
		} else { // no revive, switch pokemon
			newPokemon, err := changeActivePokemon(trainerPokemons, outChannel)
			if err != nil {
				return err
			}
			selectedPokemon = newPokemon
			return nil
		}
	}

	aux := float64(selectedPokemon.HP) / float64(selectedPokemon.MaxHP)
	var probUseItem = math.Min(math.Max(0.7, 1-aux), (1-aux)/3)

	for {
		randNr := rand.Float64()
		var probAttack = (1 - probUseItem) / 2
		var probDef = (1 - probUseItem) / 2

		if randNr < probAttack {
			// attack
			log.Info("Attacking...")
			toSend := websockets.Message{MsgType: battles.ATTACK, MsgArgs: []string{}}
			websockets.SendMessage(toSend, outChannel)
		} else if randNr < probAttack+probDef {
			// defend
			log.Info("Defending...")
			toSend := websockets.Message{MsgType: battles.DEFEND, MsgArgs: []string{}}
			websockets.SendMessage(toSend, outChannel)
		} else {
			// use item
			itemToUse, err := getItemToUseOnPokemon(trainerItems)
			if err != nil {
				probUseItem = 0
				continue
			}
			log.Info("Using item...")
			toSend := websockets.Message{MsgType: battles.USE_ITEM, MsgArgs: []string{itemToUse.Id.Hex()}}
			websockets.SendMessage(toSend, outChannel)
		}
		return nil
	}
}

func getReviveItem(trainerItems map[string]items.Item) (*items.Item, error) {

	for _, item := range trainerItems {
		if item.Effect.Appliable && item.Effect == items.ReviveEffect {
			return &item, nil
		}
	}
	return nil, errors.New("No revive item")
}

func getItemToUseOnPokemon(trainerItems map[string]items.Item) (*items.Item, error) {

	for _, item := range trainerItems {
		if item.Effect.Appliable {
			return &item, nil
		}
	}
	return nil, errors.New("No appliable items")
}

func changeActivePokemon(pokemons map[string]*pokemons.Pokemon, outChannel chan *string) (*pokemons.Pokemon, error) {
	nextPokemon, err := getAlivePokemon(pokemons)
	if err != nil {
		log.Error("No pokemons alive")
		return nil, err
	}
	log.Infof("Selecting pokemon:\tID:%s, HP: %d, Species: %s", nextPokemon.Id.Hex(),
		nextPokemon.HP,
		nextPokemon.Species)
	toSend := websockets.Message{MsgType: battles.SELECT_POKEMON, MsgArgs: []string{nextPokemon.Id.Hex()}}
	websockets.SendMessage(toSend, outChannel)
	return nextPokemon, nil
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

	rand.Seed(time.Now().UnixNano())

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

