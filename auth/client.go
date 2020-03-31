package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	auth "github.com/NOVAPokemon/authentication/exported"
	trainers "github.com/NOVAPokemon/trainers/exported"
	"github.com/NOVAPokemon/utils"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type AuthClient struct {
	Jar      *cookiejar.Jar
	Username string
	Password string
}

func (client *AuthClient) LoginWithUsernameAndPassword(username, password string) {

	httpClient := &http.Client{
		Jar: client.Jar,
	}

	jsonStr, err := json.Marshal(utils.UserJSON{Username: username, Password: password})
	if err != nil {
		log.Error(err)
	}

	host := fmt.Sprintf("%s:%d", utils.Host, utils.AuthenticationPort)
	loginUrl := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   auth.LoginPath,
	}

	req, err := http.NewRequest("POST", loginUrl.String(), bytes.NewBuffer(jsonStr))

	if err != nil {
		log.Error(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)

	log.Info(resp)

	if err != nil {
		log.Error(err)
		return
	}
}

func GetInitialTokens(username string, jar *cookiejar.Jar) error {
	httpClient := &http.Client{
		Jar: jar,
	}

	host := fmt.Sprintf("%s:%d", utils.Host, utils.TrainersPort)
	generateTokensUrl := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   fmt.Sprintf(trainers.GenerateAllTokensPath, username),
	}

	log.Info("requesting tokens at ", generateTokensUrl.String())

	req, err := http.NewRequest("GET", generateTokensUrl.String(), nil)

	if err != nil {
		log.Error(err)
		return err
	}

	resp, err := httpClient.Do(req)

	if err != nil {
		log.Error(err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("unexpected reponse")
	}

	return nil
}
