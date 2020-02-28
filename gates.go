package main

import (
	"errors"
	"math/big"
)

func Eval(circuit Circuit, shares map[PartyID]*big.Int, id PartyID) (*big.Int, error) {
	wireOutput := make(map[WireID]*big.Int)

	for _, op := range circuit {
		if ret := op.Eval(shares, id, wireOutput); ret != nil{
			return  ret, nil
		}

	}

	return nil, errors.New("no Reveal gate encountered")
}
