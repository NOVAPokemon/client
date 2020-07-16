package main

import (
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/NOVAPokemon/utils/clients"
	"github.com/NOVAPokemon/utils/items"
	"github.com/NOVAPokemon/utils/pokemons"
	"github.com/NOVAPokemon/utils/tokens"
	ws "github.com/NOVAPokemon/utils/websockets"
	"github.com/NOVAPokemon/utils/websockets/battles"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var (
	totalTimeTookStart  int64 = 0
	numberMeasuresStart       = 0

	totalTimeTookBattleMsgs  int64 = 0
	numberMeasuresBattleMsgs       = 0
)

const (
	logTimeTookStartBattle    = "time start battle: %d ms"
	logAverageTimeStartBattle = "average start battle: %f ms"

	logTimeTookBattleMsg    = "time battle: %d ms"
	logAverageTimeBattleMsg = "average battle: %f ms"
)

func autoManageBattle(trainersClient *clients.TrainersClient, conn *websocket.Conn, channels battles.BattleChannels,
	chosenPokemons map[string]*pokemons.Pokemon, requestTimestamp int64) error {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Error(err)
		}
	}()

	rand.Seed(time.Now().Unix())
	const timeout = 30 * time.Second
	cdTimer := time.NewTimer(2 * time.Second)
	expireTimer := time.NewTimer(timeout)

	var (
		startTime        = time.Now()
		started          = make(chan struct{})
		selectedPokemon  *pokemons.Pokemon
		adversaryPokemon *pokemons.Pokemon
		desMsg           ws.Serializable
	)

	for {
		select {
		case <-channels.FinishChannel:
			log.Warn("Leaving lobby because it was finished")
			return nil
		case <-channels.RejectedChannel:
			log.Warn("Leaving lobby because it was rejected")
			return nil
		case <-cdTimer.C:
			// if the battle hasn't started but the updatedPokemon is already picked, do nothing
			select {
			case <-started:
				err := doNextBattleMove(selectedPokemon, chosenPokemons, trainersClient.ItemsClaims.Items, channels.OutChannel)
				if err != nil {
					if strings.Contains(err.Error(), errorNoPokemonAlive.Error()) {
						log.Warn(err)
					} else {
						log.Error(err)
					}

					continue
				}
			default:
				remainingTime := time.Until(startTime.Add(timeout))
				log.Infof("Waiting on other player, timing out in %f seconds", remainingTime.Seconds())
			}
			cooldownDuration := time.Duration(randInt(1000, 1500))
			cdTimer.Reset(cooldownDuration * time.Millisecond)
		case msg, ok := <-channels.InChannel:
			if !ok {
				return nil
			}

			msgParsed, err := ws.ParseMessage(msg)
			if err != nil {
				return wrapAutoManageBattleError(err)
			}

			switch msgParsed.MsgType {
			case ws.Start:
				close(started)
				if !expireTimer.Stop() {
					<-expireTimer.C
				}
				if requestTimestamp == 0 {
					break
				}

				responseTime := ws.MakeTimestamp()
				timeTook := responseTime - requestTimestamp
				log.Infof(logTimeTookStartBattle, timeTook)

				numberMeasuresStart++
				totalTimeTookStart += timeTook

				log.Infof(logAverageTimeStartBattle, float64(totalTimeTookStart)/float64(numberMeasuresStart))
			case ws.Reject:
				log.Info("battle was rejected")
				close(channels.RejectedChannel)

				if requestTimestamp == 0 {
					break
				}

				responseTime := ws.MakeTimestamp()
				timeTook := responseTime - requestTimestamp
				log.Infof(logTimeTookStartBattle, timeTook)

				numberMeasuresStart++
				totalTimeTookStart += timeTook
				log.Infof(logAverageTimeStartBattle, float64(totalTimeTookStart)/float64(numberMeasuresStart))
			case ws.Error:
				desMsg, err = ws.DeserializeMsg(msgParsed)
				if err != nil {
					return wrapAutoManageBattleError(err)
				}

				errMsg := desMsg.(*ws.ErrorMessage)
				if errMsg.Fatal {
					return wrapAutoManageBattleError(newBattleErrorMsgError(errMsg.Info))
				} else {
					log.Warn(errMsg.Info)
				}
			case ws.Finish:
				log.Info("Received finish message")
				close(channels.FinishChannel)
			case battles.UpdatePokemon:
				desMsg, err = battles.DeserializeBattleMsg(msgParsed)
				if err != nil {
					return wrapAutoManageBattleError(err)
				}

				updatePokemonMsg := desMsg.(*battles.UpdatePokemonMessage)
				updatePokemonMsg.Receive(ws.MakeTimestamp())

				timeTook, valid := updatePokemonMsg.TimeTook()
				if valid {
					totalTimeTookBattleMsgs += timeTook
					numberMeasuresBattleMsgs++
					log.Infof(logTimeTookBattleMsg, timeTook)
					log.Infof(logAverageTimeBattleMsg,
						float64(totalTimeTookBattleMsgs)/float64(numberMeasuresBattleMsgs))
				}

				updatePokemonMsg.LogReceive(battles.UpdatePokemon)

				updatedPokemon := updatePokemonMsg.Pokemon
				if updatePokemonMsg.Owner {
					chosenPokemons[updatedPokemon.Id.Hex()] = &updatedPokemon
					selectedPokemon = &updatedPokemon
					log.Infof("Self Pokemon:\tID:%s, HP:%d, maxHP:%d, Species:%s", selectedPokemon.Id.Hex(),
						selectedPokemon.HP,
						selectedPokemon.MaxHP,
						selectedPokemon.Species)

				} else {
					adversaryPokemon = &updatedPokemon
					log.Infof("Adversary Pokemon:\tID:%s, HP:%d, maxHP:%d, Species:%s", adversaryPokemon.Id.Hex(),
						adversaryPokemon.HP,
						adversaryPokemon.MaxHP,
						adversaryPokemon.Species)
				}

			case battles.RemoveItem:
				desMsg, err = battles.DeserializeBattleMsg(msgParsed)
				if err != nil {
					return wrapAutoManageBattleError(err)
				}

				removeItemMsg := desMsg.(*battles.RemoveItemMessage)
				delete(trainersClient.ItemsClaims.Items, removeItemMsg.ItemId)
			case ws.SetToken:
				desMsg, err = battles.DeserializeBattleMsg(msgParsed)
				if err != nil {
					return wrapAutoManageBattleError(err)
				}

				setTokenMsg := desMsg.(*ws.SetTokenMessage)
				switch setTokenMsg.TokenField {
				case tokens.StatsTokenHeaderName:
					var statsToken *tokens.TrainerStatsToken
					statsToken, err = tokens.ExtractStatsToken(setTokenMsg.TokensString[0])
					if err != nil {
						log.Error(err)
						continue
					}
					trainersClient.TrainerStatsClaims = statsToken
					trainersClient.TrainerStatsToken = setTokenMsg.TokensString[0]

				case tokens.PokemonsTokenHeaderName:
					pokemonTkns := setTokenMsg.TokensString
					for _, tkn := range pokemonTkns {

						if len(tkn) == 0 {
							continue
						}

						var pokemonToken *tokens.PokemonToken
						pokemonToken, err = tokens.ExtractPokemonToken(tkn)
						if err != nil {
							log.Error(err)
							continue
						}

						trainersClient.ClaimsLock.Lock()

						trainersClient.PokemonClaims[pokemonToken.Pokemon.Id.Hex()] = *pokemonToken
						trainersClient.PokemonTokens[pokemonToken.Pokemon.Id.Hex()] = tkn

						trainersClient.ClaimsLock.Unlock()
					}
				case tokens.ItemsTokenHeaderName:
					var itemsToken *tokens.ItemsToken
					itemsToken, err = tokens.ExtractItemsToken(setTokenMsg.TokensString[0])
					if err != nil {
						log.Error(err)
						continue
					}
					trainersClient.ItemsClaims = itemsToken
					trainersClient.ItemsToken = setTokenMsg.TokensString[0]
				}
				log.Warn("Updated Token!")
			}
		case <-expireTimer.C:
			log.Warn("Leaving lobby because other player hasn't joined")
		}
	}
}

func doNextBattleMove(selectedPokemon *pokemons.Pokemon, trainerPokemons map[string]*pokemons.Pokemon,
	trainerItems map[string]items.Item, outChannel chan ws.Serializable) error {
	if selectedPokemon == nil {
		newPokemon, err := changeActivePokemon(trainerPokemons, outChannel)
		if err != nil {
			return wrapNextBattleMoveError(err)
		}

		selectedPokemon = newPokemon
		return nil
	}

	if selectedPokemon.HP == 0 {
		// see if we have revive
		revive, err := getReviveItem(trainerItems)
		if err != nil {
			log.Info("no revive items left")
		} else {
			log.Infof("Using revive item ID %s...", revive.Id.Hex())
			useItemMsg := battles.NewUseItemMessage(revive.Id.Hex())
			useItemMsg.Emit(ws.MakeTimestamp())
			useItemMsg.LogEmit(battles.UseItem)
			toSend := useItemMsg
			outChannel <- toSend

			if err != nil {
				return err
			}

			return nil
		} // no revive, switch pokemon

		newPokemon, err := changeActivePokemon(trainerPokemons, outChannel)
		if err != nil {
			return wrapNextBattleMoveError(err)
		}
		selectedPokemon = newPokemon
		return nil
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
			attackMsg := battles.NewAttackMessage()
			attackMsg.Emit(ws.MakeTimestamp())
			attackMsg.LogEmit(battles.Attack)
			toSend := attackMsg
			outChannel <- toSend
		} else if randNr < probAttack+probDef {
			// defend
			log.Info("Defending...")
			toSend := battles.DefendMessage{}
			outChannel <- toSend
		} else {
			// use item
			itemToUse, err := getItemToUseOnPokemon(trainerItems)
			if err != nil {
				probUseItem = 0
				continue
			}
			log.Infof("Using item: %s", itemToUse.Id.Hex())
			useItemMsg := battles.NewUseItemMessage(itemToUse.Id.Hex())
			useItemMsg.Emit(ws.MakeTimestamp())
			useItemMsg.LogEmit(battles.UseItem)
			toSend := useItemMsg
			outChannel <- toSend

			if err != nil {
				return err
			}
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

	return nil, errorNoReviveItem
}

func getItemToUseOnPokemon(trainerItems map[string]items.Item) (*items.Item, error) {
	for _, item := range trainerItems {
		if item.Effect.Appliable {
			return &item, nil
		}
	}

	return nil, errorNoAppliableItems
}

func changeActivePokemon(pokemons map[string]*pokemons.Pokemon, outChannel chan ws.Serializable) (*pokemons.Pokemon, error) {
	nextPokemon, err := getAlivePokemon(pokemons)
	if err != nil {
		return nil, wrapChangeActivePokemonError(err)
	}
	log.Infof("Selecting pokemon:\tID:%s, HP: %d, Species: %s", nextPokemon.Id.Hex(),
		nextPokemon.HP,
		nextPokemon.Species)

	selectPokemonMsg := battles.NewSelectPokemonMessage(nextPokemon.Id.Hex())
	selectPokemonMsg.Emit(ws.MakeTimestamp())
	selectPokemonMsg.LogEmit(battles.SelectPokemon)

	toSend := selectPokemonMsg
	outChannel <- toSend

	return nextPokemon, nil
}

func getAlivePokemon(pokemons map[string]*pokemons.Pokemon) (*pokemons.Pokemon, error) {
	for _, v := range pokemons {
		if v.HP > 0 {
			return v, nil
		}
	}

	return nil, errorNoPokemonAlive
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}

	return string(s)
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}
