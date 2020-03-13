package main

import (
	"fmt"
	"math/big"
)

type WireID uint64

type GateID uint64

type Operation interface {
	Output() WireID
	Eval(MPCProtocol *Protocol, shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int
}

type Input struct {
	Party PartyID
	Out   WireID
}

func (io Input) Output() WireID {
	return io.Out
}

func (io Input) Eval(MPCProtocol *Protocol, shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int {
	wireOutput[io.Out] = shares[io.Party]
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
func (ao Add) Eval(MPCProtocol *Protocol, shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int {
	fmt.Println("Salut", ao.In1, ao.In2, wireOutput, id)
	wireOutput[ao.Out] = new(big.Int).Add(wireOutput[ao.In1], wireOutput[ao.In2])
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
func (aco AddCst) Eval(MPCProtocol *Protocol, shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int {
	wireOutput[aco.Out] = big.NewInt(wireOutput[aco.In].Int64())
	if id == 0 {
		wireOutput[aco.Out].Add(wireOutput[aco.Out], big.NewInt(int64(aco.CstValue)))
	}
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

func (so Sub) Eval(MPCProtocol *Protocol, shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int {
	wireOutput[so.Out] = new(big.Int).Sub(wireOutput[so.In1], wireOutput[so.In2])
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
func (mo Mult) Eval(MPCProtocol *Protocol, shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int {
	query := BeaverMessage{}
	query.PartyID = id
	query.PartyCount = uint64(len(MPCProtocol.Peers))
	query.In1 = mo.In1
	query.In2 = mo.In2
	query.Out = mo.Out

	MPCProtocol.ThirdPartyChans.Send <- query
	response := <-MPCProtocol.ThirdPartyChans.Receive

	fmt.Println(response)

	return nil
}

type MultCst struct {
	In       WireID
	CstValue uint64
	Out      WireID
}

func (mco MultCst) Output() WireID {
	return mco.Out
}

func (mco MultCst) Eval(MPCProtocol *Protocol, shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int {
	wireOutput[mco.Out] = new(big.Int).Mul(wireOutput[mco.In], big.NewInt(int64(mco.CstValue)))
	return nil
}

type Reveal struct {
	In  WireID
	Out WireID
}

func (ro Reveal) Output() WireID {
	return ro.Out
}
func (ro Reveal) Eval(MPCProtocol *Protocol, shares map[PartyID]*big.Int, id PartyID, wireOutput map[WireID]*big.Int) *big.Int {
	return wireOutput[ro.In].Mod(wireOutput[ro.In], q)
}
