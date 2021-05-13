package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
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

	rlimit := unix.Rlimit{}
	err = unix.Getrlimit(unix.RLIMIT_NPROC, &rlimit)
	if err != nil {
		log.Panic("error setting limit in OS: %v", rlimit)
	}

	log.Printf("current rlimit %+v", rlimit)

	rlimit.Cur = 100000
	rlimit.Max = 100000

	err = unix.Setrlimit(unix.RLIMIT_NPROC, &rlimit)
	if err != nil {
		log.Panic("error setting limit in OS: %v", rlimit)
	}

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

	randomTime := time.Duration(rand.Intn(10)) * time.Second
	time.Sleep(randomTime)

	filename := fmt.Sprintf("%s/client_%d.log", logsDir, clientNum)
	file, err := os.Create(filename)
	if err != nil {
		log.Panic(fmt.Errorf("error in client %d creating file: %w", clientNum, err))
	}

	args := []string{
		"-a",
		"-n", strconv.Itoa(clientNum),
		"-r", clientsRegion,
		"-t", clientTimeout,
		"-ld", logsDir,
	}
	cmd := exec.Command(fmt.Sprintf("%s/client/executable", projectDir), args...)
	cmd.Stdout = file
	cmd.Stderr = file

	err = cmd.Run()
	if err != nil {
		log.Error(fmt.Errorf("error in client %d running: %w", clientNum, err))

		err = file.Sync()
		if err != nil {
			log.Error(fmt.Errorf("error in client %d syncing file: %w", clientNum, err))
		}
	}
}
