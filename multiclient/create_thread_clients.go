package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	numClientsEnvVarName     = "NUM_CLIENTS"
	regionEnvVarName         = "REGION"
	clientsTimeoutEnvVarName = "CLIENTS_TIMEOUT"
	logsDirEnvVarName        = "LOGS_DIR"
	novapokemonDirEnvVarName = "NOVAPOKEMON"
)

func main() {
	numClients, err := strconv.Atoi(getValidEnvVariable(numClientsEnvVarName))
	if err != nil {
		log.Panic(err)
	}

	clientsRegion := getValidEnvVariable(regionEnvVarName)
	clientTimeout := getValidEnvVariable(clientsTimeoutEnvVarName)
	logsDir := getValidEnvVariable(logsDirEnvVarName)
	projectDir := getValidEnvVariable(novapokemonDirEnvVarName)

	wg := &sync.WaitGroup{}

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go launchClient(wg, clientsRegion, clientTimeout, logsDir, projectDir, i)
	}

	wg.Wait()
	log.Infof("Finished clients")
}

func getValidEnvVariable(envVarName string) string {
	if varValue := os.Getenv(envVarName); varValue == "" {
		log.Panicf("%s is empty", envVarName)
		return ""
	} else {
		return varValue
	}
}

func launchClient(wg *sync.WaitGroup, clientsRegion, clientTimeout, logsDir, projectDir string, clientNum int) {
	defer wg.Done()

	filename := fmt.Sprintf("%s/client_%d.log", logsDir, clientNum)
	file, err := os.Create(filename)
	if err != nil {
		log.Panic(fmt.Errorf("error in client %d creating file: %w", clientNum, err))
	}

	cmd := exec.Cmd{
		Path: fmt.Sprintf("%s/client/executable", projectDir),
		Args: []string{
			"-a",
			"-n", strconv.Itoa(clientNum),
			"-r", clientsRegion,
			"-t", clientTimeout,
			"-ld", logsDir,
		},
		Stdout: file,
		Stderr: file,
	}

	err = cmd.Run()
	if err != nil {
		log.Error(fmt.Errorf("error in client %d running: %w", clientNum, err))

		var out []byte
		out, err = cmd.CombinedOutput()
		if err != nil {
			log.Panic(fmt.Errorf("error in client %d getting output: %w", clientNum, err))
		}

		log.Warnf("%s", out)

		err = file.Sync()
		if err != nil {
			log.Error(fmt.Errorf("error in client %d syncing file: %w", clientNum, err))
		}
	}
}
