package main

func main() {

	client := NovaPokemonClient{
		Username: requestUsername(),
		Password: requestPassword(),
	}

	client.init()
	_ = LoginAndStartAutoBattleQueue(&client)

}

/*
func main2() {
	client := NovaPokemonClient{
		Username: requestUsername(),
		Password: requestPassword(),
	}
	client.init()

	client.Login()
	err := client.GetAllTokens()
	if err != nil {
		log.Error(err)
		return
	}

	for _, cookie := range client.jar.Cookies(&url.URL{
		Scheme: "http",
		Host:   "localhost",
	}) {
		log.Info(cookie)
	}

	trades := client.tradesClient.GetAvailableLobbies()
	log.Infof("Available Lobbies: %+v", trades)

	if len(trades) == 0 {
		client.tradesClient.CreateTradeLobby()
	} else {
		lobby := trades[0]
		log.Infof("Joining lobby %s", lobby)
		client.tradesClient.JoinTradeLobby(lobby.Id)
	}
}
*/
