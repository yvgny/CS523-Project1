package main

import (
	"errors"
	"math/big"
)

func Eval(circuit Circuit, shares map[PartyID]*big.Int, id PartyID) (*big.Int, error) {
	wireOutput := make(map[WireID]*big.Int)

	for _, op := range circuit {

		switch v := op.(type) {
		case Add:
			wireOutput[v.Out] = new(big.Int).Add(wireOutput[v.In1], wireOutput[v.In2])
		case Sub:
			wireOutput[v.Out] = new(big.Int).Sub(wireOutput[v.In1], wireOutput[v.In2])
		case MultCst:
			wireOutput[v.Out] = new(big.Int).Mul(wireOutput[v.In], big.NewInt(int64(v.CstValue)))
		case Input:
			wireOutput[v.Out] = shares[v.Party]
		case AddCst:
			wireOutput[v.Out] = big.NewInt(wireOutput[v.In].Int64())
			if id == 0 {
				wireOutput[v.Out].Add(wireOutput[v.Out], big.NewInt(int64(v.CstValue)))
			}
		case Reveal:
			return wireOutput[v.In], nil
		default:
			panic("Operation does not exists.")
		}

	}

	return nil, errors.New("no Reveal gate encountered")
}
