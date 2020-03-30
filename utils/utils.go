package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	auth "github.com/NOVAPokemon/authentication/exported"
	"github.com/NOVAPokemon/utils"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
)

func Login(jar *cookiejar.Jar) {
	username := requestUsername()
	password := requestPassword()

	LoginWithUsernameAndPassword(username, password, jar)
}

func LoginWithUsernameAndPassword(username, password string, jar *cookiejar.Jar) {

	httpClient := &http.Client{
		Jar: jar,
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

	for _, cookie := range jar.Cookies(&loginUrl) {
		log.Info(cookie)
	}
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
