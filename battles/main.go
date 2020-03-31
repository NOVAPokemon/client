package battles

func main() {
	/*
		jar, err := cookiejar.New(nil)

		if err != nil {
			log.Error(err)
			return
		}

		var hostAddr = fmt.Sprintf("%s:%d", utils.Host, utils.BattlesPort)

		client := BattleLobbyClient{
			jar: &cookiejar.jar{

			},
			BattlesAddr: hostAddr,
		}

		lobbyClient := client.NewBattleLobbyClient(hostAddr, utils.Trainer{}, nil)
		battles := client.GetAvailableLobbies(lobbyClient)

		log.Infof("Available Lobbies: %+v", battles)

		if len(battles) == 0 {
			// login
			clientUtils.LoginWithUsernameAndPassword("trainer1", "qwe", jar)
			lobbyClient = NewBattleLobbyClient(hostAddr, utils.Trainer{}, jar)

			// create new lobby
			CreateBattleLobby(lobbyClient)

		} else {
			// login
			clientUtils.LoginWithUsernameAndPassword("trainer2", "qwe", jar)
			lobbyClient = NewBattleLobbyClient(hostAddr, utils.Trainer{}, jar)

			// join client
			lobby := battles[0]
			log.Infof("Joining lobby %s", lobby)
			JoinBattleLobby(lobbyClient, lobby.Id)
		}
	*/
}
