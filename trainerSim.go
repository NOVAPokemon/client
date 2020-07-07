package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

type operation string

// *** WARNING ***
// ORDER MATTERS FOR MATRIX
var ops = []operation{challengeCmd, queueCmd, makeMicrotransactionCmd, tradeCmd, storeCmd, catchCmd, raidCmd}

const (
	challengeCmd                operation = "b"
	challengeSpecificTrainerCmd operation = "bs"
	queueCmd                    operation = "q"
	tradeSpecificTrainerCmd     operation = "ts"
	tradeCmd                    operation = "t"
	makeMicrotransactionCmd     operation = "m"
	storeCmd                    operation = "s"
	catchCmd                    operation = "c"
	raidCmd                     operation = "r"
	noOp                        operation = "h"
	exitCmd                     operation = "e"

	acceptCmd operation = "a"
	rejectCmd operation = "r"
)

type trainerSim struct {
	previousOp int
	ProbMatrix [][]float32
}

func newTrainerSim() *trainerSim {
	rand.Seed(time.Now().Unix())
	return &trainerSim{
		ProbMatrix: getAndVerifyProbMatrix(),
	}
}

func getAndVerifyProbMatrix() [][]float32 {
	/*
		data, err := ioutil.ReadFile("./probability_matrix.json")
		if err != nil {
			log.Fatal("Error reading file: ", err)
		}

		matrix := make([][]float32, len(ops))
		err = json.Unmarshal(data, &matrix)
		if err != nil {
			log.Fatal("Error parsing matrix file: ", err)
		}

		if len(matrix) != len(ops) {
			panic("Matrix has wrong number of lines")
		}
	*/
	matrixAux := matrix

	for i := 0; i < len(matrixAux); i++ {
		probs := matrixAux[i]
		var total float32

		if len(probs) != len(ops) {
			panic(fmt.Sprintf("Line %d has incorrect number of rows", i))
		}

		for j := 0; j < len(probs); j++ {
			total += probs[j]
		}
		if math.Abs(float64(1-total)) > 0.01 { // give a margin of 1% error for manual config of probabilities
			panic(fmt.Sprintf("Probabilities of row corresponding to %s (%d) do not sum to 1", string(ops[i]), i))
		}
	}

	return matrixAux
}

func (s *trainerSim) getNextOperation() operation {

	s.logNextProbabilities()

	nextRand := rand.Float32()
	log.Infof("Chose random: %f", nextRand)
	probs := s.ProbMatrix[s.previousOp]

	var aux float32 = 0
	for i := 0; i < len(probs); i++ {
		aux += probs[i]
		if nextRand < aux {
			log.Infof("chose operation : %s", string(ops[i]))
			s.previousOp = i
			return ops[i]
		}
	}
	panic("Did not get any move from matrix of operations, probabilities are probably set up wrong")
}

func (s *trainerSim) logNextProbabilities() {
	probs := s.ProbMatrix[s.previousOp]
	log.Infof("Previous operation : %s", string(ops[s.previousOp]))
	for i := 0; i < len(probs); i++ {
		log.Infof("Probability to %s : %f", string(ops[i]), probs[i])
	}

}
