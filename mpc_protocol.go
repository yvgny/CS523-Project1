package main

import (
	"fmt"
	"math/big"

	"github.com/ldsec/lattigo/bfv"
	"github.com/ldsec/lattigo/ring"
)

var q = ring.NewUint(bfv.DefaultParams[bfv.PN13QP218].T)

// Structure of network message to carry the output value of a specific wire
type MPCMessage struct {
	Out   WireID
	Value uint64
}

type Protocol struct {
	*LocalParty

	Input          uint64
	Output         uint64
	Circuit        Circuit
	WireOutput     map[WireID]*big.Int      // store each the output of each wire
	BeaverTriplets map[WireID]BeaverTriplet // store the triplet used for each multiplication gate
}

// Create a new protocol to compute the value produced by 'Circuit' when fed with 'input'. The number of beaver triplets given must be >= to the number of multiplication gate present in the circuit
func (lp *LocalParty) NewProtocol(input uint64, circuit Circuit, beaverTriplets map[WireID]BeaverTriplet) *Protocol {
	cep := new(Protocol)
	cep.LocalParty = lp
	cep.WireOutput = make(map[WireID]*big.Int)
	cep.BeaverTriplets = beaverTriplets
	cep.Circuit = circuit

	cep.Input = input
	return cep
}

// Start the circuit computation
func (cep *Protocol) Run() {

	fmt.Println(cep, "is running")

	for _, op := range cep.Circuit {
		op.Eval(cep)
	}

	cep.Output = cep.WireOutput[cep.Circuit[len(cep.Circuit)-1].Output()].Uint64()

	if cep.WaitGroup != nil {
		cep.WaitGroup.Done()
	}

}
