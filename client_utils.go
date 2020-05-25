package main

import (
	"github.com/NOVAPokemon/utils/clients"
	"github.com/NOVAPokemon/utils/items"
	"github.com/NOVAPokemon/utils/pokemons"
	"github.com/NOVAPokemon/utils/tokens"
	ws "github.com/NOVAPokemon/utils/websockets"
	"github.com/NOVAPokemon/utils/websockets/battles"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"math"
	"math/rand"
	"time"
)

var (
	totalTimeTookBattleMsgs  int64 = 0
	numberMeasuresBattleMsgs       = 0
)

func autoManageBattle(trainersClient *clients.TrainersClient, conn *websocket.Conn, channels battles.BattleChannels,
	chosenPokemons map[string]*pokemons.Pokemon) error {
	defer ws.CloseConnection(conn)

	rand.Seed(time.Now().Unix())
	const timeout = 30 * time.Second

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
			ws.CloseConnection(conn)
		}
	}()

	for {
		select {

		case <-channels.FinishChannel:
			return nil

		case <-cdTimer.C:
			// if the battle hasn't started but the updatedPokemon is already picked, do nothing
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

			msgParsed, err := ws.ParseMessage(msg)
			if err != nil {
				return wrapAutoManageBattleError(err)
			}

			switch msgParsed.MsgType {
			case ws.Start:
				started = true

			case ws.Error:
				desMsg, err := ws.DeserializeMsg(msgParsed)
				if err != nil {
					return wrapAutoManageBattleError(err)
				}

				errMsg := desMsg.(*ws.ErrorMessage)
				if errMsg.Fatal {
					return wrapAutoManageBattleError(newBattleErrorMsgError(errMsg.Info))
				} else {
					log.Warn(errMsg.Info)
				}

			case battles.UpdatePokemon:
				desMsg, err := battles.DeserializeBattleMsg(msgParsed)
				if err != nil {
					return wrapAutoManageBattleError(err)
				}

				updatePokemonMsg := desMsg.(*battles.UpdatePokemonMessage)
				updatePokemonMsg.Receive(ws.MakeTimestamp())

				timeTook, ok := updatePokemonMsg.TimeTook()
				if ok {
					totalTimeTookBattleMsgs += timeTook
					numberMeasuresBattleMsgs++
					log.Infof("time took: %d ms", timeTook)
					log.Infof("average time for battle msgs: %f ms",
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
				desMsg, err := battles.DeserializeBattleMsg(msgParsed)
				if err != nil {
					return wrapAutoManageBattleError(err)
				}

				removeItemMsg := desMsg.(*battles.RemoveItemMessage)
				delete(trainersClient.ItemsClaims.Items, removeItemMsg.ItemId)
			case ws.SetToken:
				desMsg, err := battles.DeserializeBattleMsg(msgParsed)
				if err != nil {
					return wrapAutoManageBattleError(err)
				}

				setTokenMsg := desMsg.(*ws.SetTokenMessage)
				switch setTokenMsg.TokenField {
				case tokens.StatsTokenHeaderName:
					decodedToken, err := tokens.ExtractStatsToken(setTokenMsg.TokensString[0])
					if err != nil {
						log.Error(err)
						continue
					}
					trainersClient.TrainerStatsClaims = decodedToken
					trainersClient.TrainerStatsToken = setTokenMsg.TokensString[0]

				case tokens.PokemonsTokenHeaderName:
					pokemonTkns := setTokenMsg.TokensString
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
					decodedToken, err := tokens.ExtractItemsToken(setTokenMsg.TokensString[0])
					if err != nil {
						log.Error(err)
						continue
					}
					trainersClient.ItemsClaims = decodedToken
					trainersClient.ItemsToken = setTokenMsg.TokensString[0]
				}
				log.Warn("Updated Token!")
			}
		}
	}
}

func doNextBattleMove(selectedPokemon *pokemons.Pokemon, trainerPokemons map[string]*pokemons.Pokemon, trainerItems map[string]items.Item, outChannel chan ws.GenericMsg) error {
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
			toSend := useItemMsg.SerializeToWSMessage()
			outChannel <- ws.GenericMsg{
				MsgType: websocket.TextMessage,
				Data:    []byte(toSend.Serialize()),
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
			toSend := attackMsg.SerializeToWSMessage()
			outChannel <- ws.GenericMsg{
				MsgType: websocket.TextMessage,
				Data:    []byte(toSend.Serialize()),
			}
		} else if randNr < probAttack+probDef {
			// defend
			log.Info("Defending...")
			toSend := battles.DefendMessage{}.SerializeToWSMessage()
			outChannel <- ws.GenericMsg{
				MsgType: websocket.TextMessage,
				Data:    []byte(toSend.Serialize()),
			}
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
			toSend := useItemMsg.SerializeToWSMessage()
			outChannel <- ws.GenericMsg{
				MsgType: websocket.TextMessage,
				Data:    []byte(toSend.Serialize()),
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

func changeActivePokemon(pokemons map[string]*pokemons.Pokemon, outChannel chan ws.GenericMsg) (*pokemons.Pokemon, error) {
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

	toSend := selectPokemonMsg.SerializeToWSMessage()
	outChannel <- ws.GenericMsg{
		MsgType: websocket.TextMessage,
		Data:    []byte(toSend.Serialize()),
	}
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

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}

	return string(s)
}

func RandInt(min int, max int) int {
	return min + rand.Intn(max-min)
}
