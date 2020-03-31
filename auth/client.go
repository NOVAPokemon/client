package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	auth "github.com/NOVAPokemon/authentication/exported"
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
