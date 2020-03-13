package main

import (
	"fmt"
	"math/big"

	"github.com/ldsec/lattigo/ring"
)

type WireID uint64

type GateID uint64

type Operation interface {
	Output() WireID
	Eval(shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int
	GenerateTriplet(count int) *BeaverTriplet
}

type Input struct {
	Party PartyID
	Out   WireID
}

func (io Input) Output() WireID {
	return io.Out
}

func (io Input) Eval(shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int {
	wireOutput[io.Out] = shares[io.Party]
	return nil
}

func (io Input) GenerateTriplet(count int) *BeaverTriplet {
	return nil
}

type Add struct {
	In1 WireID
	In2 WireID
	Out WireID
}

func (ao Add) Output() WireID {
	return ao.Out
}
func (ao Add) Eval(shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int {
	fmt.Println("Salut", ao.In1, ao.In2, wireOutput, id)
	wireOutput[ao.Out] = new(big.Int).Add(wireOutput[ao.In1], wireOutput[ao.In2])
	return nil
}
func (ao Add) GenerateTriplet(count int) *BeaverTriplet {
	return nil
}

type AddCst struct {
	In       WireID
	CstValue uint64
	Out      WireID
}

func (aco AddCst) Output() WireID {
	return aco.Out
}
func (aco AddCst) Eval(shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int {
	wireOutput[aco.Out] = big.NewInt(wireOutput[aco.In].Int64())
	if id == 0 {
		wireOutput[aco.Out].Add(wireOutput[aco.Out], big.NewInt(int64(aco.CstValue)))
	}
	return nil
}

func (aco AddCst) GenerateTriplet(count int) *BeaverTriplet {
	return nil
}

type Sub struct {
	In1 WireID
	In2 WireID
	Out WireID
}

func (so Sub) Output() WireID {
	return so.Out
}

func (so Sub) Eval(shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int {
	wireOutput[so.Out] = new(big.Int).Sub(wireOutput[so.In1], wireOutput[so.In2])
	return nil
}
func (so Sub) GenerateTriplet(count int) *BeaverTriplet {
	return nil
}

type Mult struct {
	In1 WireID
	In2 WireID
	Out WireID
}

func (mo Mult) Output() WireID {
	return mo.Out
}
func (mo Mult) Eval(shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int {
	panic("Not implemented")
}

func (mo Mult) GenerateTriplet(count int) *BeaverTriplet {
	a := ring.RandInt(q)
	b := ring.RandInt(q)
	triplet := &BeaverTriplet{
		In1: mo.In1,
		In2: mo.In2,
		Out: mo.Out,
		a:   a,
		b:   b,
		c:   big.NewInt(0).Mul(a, b),
	}
	return triplet
}

type MultCst struct {
	In       WireID
	CstValue uint64
	Out      WireID
}

func (mco MultCst) Output() WireID {
	return mco.Out
}

func (mco MultCst) Eval(shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int {
	wireOutput[mco.Out] = new(big.Int).Mul(wireOutput[mco.In], big.NewInt(int64(mco.CstValue)))
	return nil
}

func (mco MultCst) GenerateTriplet(count int) *BeaverTriplet {
	return nil
}

type Reveal struct {
	In  WireID
	Out WireID
}

func (ro Reveal) Output() WireID {
	return ro.Out
}
func (ro Reveal) Eval(shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int {
	return wireOutput[ro.In].Mod(wireOutput[ro.In], q)
}
func (ro Reveal) GenerateTriplet(count int) *BeaverTriplet {
	return nil
}
