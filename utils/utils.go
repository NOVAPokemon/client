package utils

import (
	"bufio"
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
	"os"
	"strings"
)

func Login(jar *cookiejar.Jar) (string, error) {
	username := requestUsername()
	password := requestPassword()

	return username, LoginWithUsernameAndPassword(username, password, jar)
}

func LoginWithUsernameAndPassword(username, password string, jar *cookiejar.Jar) error {
	httpClient := &http.Client{
		Jar: jar,
	}

	jsonStr, err := json.Marshal(utils.UserJSON{Username: username, Password: password})
	if err != nil {
		log.Error(err)
		return err
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
		return err
	}

	req.Header.Set("Content-Type", "application/json")

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
