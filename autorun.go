package main

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
)

const filename = "actions.json"

type Actions struct {
	Battle int
	Trade  int
	Store  int
	Catch  int
}

func writeAutoRunConfigFile(actions *Actions) {
	file, _ := json.MarshalIndent(actions, "", " ")

	_ = ioutil.WriteFile(filename, file, 0644)
}

func readAutoRunConfigFile() *Actions {
	jsonBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Error(err)
		return nil
	}

	actions := &Actions{}
	err = json.Unmarshal(jsonBytes, actions)
	if err!= nil {
		log.Error(err)
		return nil
	}

	return actions
}
