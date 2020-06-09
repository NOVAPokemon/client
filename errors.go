package main

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	errorStartTrade               = "error starting trade"
	errorJoinTrade                = "error joining trade"
	errorRejectTrade              = "error rejecting trade"
	errorRegisterAndGetTokens     = "error registering and getting tokens"
	errorLoginAndGetTokens        = "error logging in and getting tokens"
	errorBuyRandomItem            = "error buying random item"
	errorCatchingWildPokemon      = "error catching wild pokemon"
	errorStartAutoChallenge       = "error starting auto challenge"
	errorChallengePlayer          = "error challenging player"
	errorStartAutoTrade           = "error starting auto trade"
	errorHandleTradeNotification  = "error handling trade notification"
	errorHandleBattleNotification = "error handling battle notification"
	errorStartAutoBattleQueue     = "error starting auto queue for battle"
	errorStartLookRaid            = "error starting to look for nearby raid"
	errorAutoManageBattle         = "error auto managing battle"
	errorNextBattleMove           = "error in next battle move"
	errorChangeActivePokemon      = "error changing active pokemon"
	errorMakeMicrotransaction     = "error making microtransaction"
	errRefreshingAuthToken        = "error refreshing auth token"

	errorBattleErrorMsgFormat = "got battle error msg %s"

	errListeningToNotifications = "error listening to notifications"
	errUpdatingLocation         = "error updating location"
)

var (
	errorInvalidCommand    = errors.New("invalid command")
	errorNotEnoughPokemons = errors.New("not enough alive pokemons to battle")
	errorNoReviveItem      = errors.New("no revive item")
	errorNoAppliableItems  = errors.New("no appliable items")
	errorNoPokemonAlive    = errors.New("no pokemons alive")
)

// Error wrappers
func wrapStartTradeError(err error) error {
	return errors.Wrap(err, errorStartTrade)
}

func wrapJoinTradeError(err error) error {
	return errors.Wrap(err, errorJoinTrade)
}

func wrapRejectTradeError(err error) error {
	return errors.Wrap(err, errorRejectTrade)
}

func wrapRegisterAndGetTokensError(err error) error {
	return errors.Wrap(err, errorRegisterAndGetTokens)
}

func wrapLoginAndGeTokensError(err error) error {
	return errors.Wrap(err, errorLoginAndGetTokens)
}

func wrapErrorListeningToNotifications(err error) error {
	return errors.Wrap(err, errListeningToNotifications)
}

func wrapErrorUpdatingLocation(err error) error {
	return errors.Wrap(err, errUpdatingLocation)
}

func wrapErrorRefreshingAuthToken(err error) error {
	return errors.Wrap(err, errRefreshingAuthToken)
}

func wrapBuyRandomItemError(err error) error {
	return errors.Wrap(err, errorBuyRandomItem)
}

func wrapMakeRandomMicrotransaction(err error) error {
	return errors.Wrap(err, errorMakeMicrotransaction)
}

func wrapCatchWildPokemonError(err error) error {
	return errors.Wrap(err, errorCatchingWildPokemon)
}

func wrapStartAutoChallengeError(err error) error {
	return errors.Wrap(err, errorStartAutoChallenge)
}

func wrapChallengePlayerError(err error) error {
	return errors.Wrap(err, errorChallengePlayer)
}

func wrapStartAutoTrade(err error) error {
	return errors.Wrap(err, errorStartAutoTrade)
}

func wrapHandleTradeNotificationError(err error) error {
	return errors.Wrap(err, errorHandleTradeNotification)
}

func wrapHandleBattleNotificationError(err error) error {
	return errors.Wrap(err, errorHandleBattleNotification)
}

func wrapStartAutoBattleQueueError(err error) error {
	return errors.Wrap(err, errorStartAutoBattleQueue)
}

func wrapStartLookForRaid(err error) error {
	return errors.Wrap(err, errorStartLookRaid)
}

func wrapAutoManageBattleError(err error) error {
	return errors.Wrap(err, errorAutoManageBattle)
}

func wrapNextBattleMoveError(err error) error {
	return errors.Wrap(err, errorNextBattleMove)
}

func wrapChangeActivePokemonError(err error) error {
	return errors.Wrap(err, errorChangeActivePokemon)
}

// Error builders
func newBattleErrorMsgError(errorMsg string) error {
	return errors.New(fmt.Sprintf(errorBattleErrorMsgFormat, errorMsg))
}
