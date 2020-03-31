package trades

func main() {
	/*
			jar, err := cookiejar.New(nil)

			if err != nil {
				log.Error(err)
				return
			}

		username, err := clientUtils.Login(jar)
		if err != nil {
			log.Error(err)
			return
		}

		err = clientUtils.GetInitialTokens(username, jar)
		if err != nil {
			log.Error(err)
			return
		}

		for _, cookie := range jar.Cookies(&url.URL{
			Scheme:     "http",
			Host:       "localhost",
		}) {
			log.Info(cookie)
		}
			auth.Login(jar)

			var hostAddr = fmt.Sprintf("%s:%d", utils.Host, utils.TradesPort)
			lobbyClient := NewTradeLobbyClient(hostAddr, utils.Trainer{}, jar)

			battles := GetAvailableLobbies(lobbyClient)
			log.Infof("Available Lobbies: %+v", battles)

			print(len(battles))

			if len(battles) == 0 {
				CreateTradeLobby(lobbyClient)
			} else {
				lobby := battles[0]
				log.Infof("Joining lobby %s", lobby)
				JoinTradeLobby(lobbyClient, lobby.Id)
			}
	*/
}
