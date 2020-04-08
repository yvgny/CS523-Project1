package main

import (
	"fmt"
	"math/big"

	"github.com/ldsec/lattigo/bfv"
	"github.com/ldsec/lattigo/ring"
)

var q = ring.NewUint(bfv.DefaultParams[bfv.PN13QP218].T)

type MPCMessage struct {
	Out   WireID
	Value uint64
}

type Protocol struct {
	*LocalParty

	Input          uint64
	Output         uint64
	Circuit        Circuit
	WireOutput     map[WireID]*big.Int
	BeaverTriplets map[WireID]BeaverTriplet
}

func (lp *LocalParty) NewProtocol(input uint64, circuit Circuit, beaverTriplets map[WireID]BeaverTriplet) *Protocol {
	cep := new(Protocol)
	cep.LocalParty = lp
	cep.WireOutput = make(map[WireID]*big.Int)
	cep.BeaverTriplets = beaverTriplets
	cep.Circuit = circuit

	cep.Input = input
	return cep
}

func (cep *Protocol) Run() {

	fmt.Println(cep, "is running")

	for _, op := range cep.Circuit {
		op.Eval(cep)
	}

	fmt.Println("for loop end")

	cep.Output = cep.WireOutput[cep.Circuit[len(cep.Circuit)-1].Output()].Uint64()

	if cep.WaitGroup != nil {
		cep.WaitGroup.Done()
	}

}
