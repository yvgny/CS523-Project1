package main

import (
	"errors"
	"fmt"
	"math/big"
)

func Eval(MPCProtocol *Protocol, circuit Circuit, shares map[PartyID]*big.Int, id PartyID) (*big.Int, error) {
	wireOutput := make(map[WireID]*big.Int)

	for _, op := range circuit {
		fmt.Println(op)
		if ret := op.Eval(MPCProtocol, shares, id, wireOutput); ret != nil {
			return ret, nil
		}

	}

	return nil, errors.New("no Reveal gate encountered")
}
