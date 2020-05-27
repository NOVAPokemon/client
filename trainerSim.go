package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"math"
	"math/rand"
	"time"
)

type Operation string

// *** WARNING ***
// ORDER MATTERS FOR MATRIX
var ops = []Operation{ChallengeCmd, QueueCmd, MakeMicrotransactionCmd, TradeCmd, StoreCmd, CatchCmd, RaidCmd}

const (
	ChallengeCmd                Operation = "b"
	ChallengeSpecificTrainerCmd Operation = "bs"
	QueueCmd                    Operation = "q"
	TradeSpecificTrainerCmd     Operation = "ts"
	TradeCmd                    Operation = "t"
	MakeMicrotransactionCmd     Operation = "m"
	StoreCmd                    Operation = "s"
	CatchCmd                    Operation = "c"
	RaidCmd                     Operation = "r"
	NoOp                        Operation = "h"
	ExitCmd                     Operation = "e"

	AcceptCmd Operation = "a"
	RejectCmd Operation = "r"
)

type TrainerSim struct {
	previousOp int
	ProbMatrix [][]float32
}

func NewTrainerSim() *TrainerSim {
	rand.Seed(time.Now().Unix())
	return &TrainerSim{
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
	matrix := Matrix

	for i := 0; i < len(matrix); i++ {
		probs := matrix[i]
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

	return matrix
}

func (s *TrainerSim) GetNextOperation() Operation {

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

func (s *TrainerSim) logNextProbabilities() {
	probs := s.ProbMatrix[s.previousOp]
	log.Infof("Previous operation : %s", string(ops[s.previousOp]))
	for i := 0; i < len(probs); i++ {
		log.Infof("Probability to %s : %f", string(ops[i]), probs[i])
	}

}
