package main

import (
	"fmt"
	"github.com/NOVAPokemon/utils/tokens"
	log "github.com/sirupsen/logrus"
	"math"
	"math/rand"
	"time"
)

type Operation rune

var ops = []Operation{ChallengeCmd, QueueCmd, TradeCmd, StoreCmd, CatchCmd}

const (
	// *** WARNING ***
	// ORDER MATTERS FOR MATRIX
	ChallengeCmd Operation = 'b'
	QueueCmd     Operation = 'q'
	TradeCmd     Operation = 't'
	StoreCmd     Operation = 's'
	CatchCmd     Operation = 'c'
	ExitCmd      Operation = 'e'
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

func (s *TrainerSim) GetNextOperation(trainer *tokens.TrainerStatsToken, pokemons map[string]tokens.PokemonToken, items *tokens.ItemsToken) Operation {

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
