package main

import (
	"crypto/rand"
	"errors"
	"math/big"
)

type WireID uint64

type GateID uint64

type BeaverTriplet struct {
	a *big.Int
	b *big.Int
	c *big.Int
}

type Operation interface {
	Output() WireID
	Eval(*Protocol)         // computes the operation of the wire and stores the result in the WireOutput map
	BeaverTriplet(int) bool // returns true if and only if beaver triplets are needed for this gate
}

// Given an input, split it between the peers and send them their share
func (io Input) generateShares(cep *Protocol) {
	sum := big.NewInt(0)
	for _, peer := range cep.Peers {
		if peer.ID != cep.ID {

			share, err := rand.Int(rand.Reader, q)
			check(err)

			sum.Add(sum, share)
			peer.Chan <- Message{MPCMessage: &MPCMessage{io.Out, share.Uint64()}}
		}
	}
	s := big.NewInt(int64(cep.Input))
	cep.WireOutput[io.Out] = new(big.Int).Sub(s, sum)
	cep.WireOutput[io.Out].Mod(cep.WireOutput[io.Out], q)
}

type Input struct {
	Party PartyID
	Out   WireID
}

func (io Input) Output() WireID {
	return io.Out
}

// If the input is our, split it using the method 'generateShares', otherwise receive our share from the concerned peer
func (io Input) Eval(cep *Protocol) {
	if io.Party == cep.ID {
		io.generateShares(cep)
	} else {
		m := <-cep.Peers[io.Party].ReceiveChan
		if m.MPCMessage == nil {
			check(errors.New("BeaverMessage received instead of MPCMessage"))
		}
		cep.WireOutput[io.Out] = big.NewInt(int64(m.MPCMessage.Value))
	}
}

func (io Input) BeaverTriplet(count int) bool {
	return false
}

type Add struct {
	In1 WireID
	In2 WireID
	Out WireID
}

func (ao Add) Output() WireID {
	return ao.Out
}

func (ao Add) Eval(cep *Protocol) {
	cep.WireOutput[ao.Out] = new(big.Int).Add(cep.WireOutput[ao.In1], cep.WireOutput[ao.In2])
}

func (ao Add) BeaverTriplet(count int) bool {
	return false
}

type AddCst struct {
	In       WireID
	CstValue uint64
	Out      WireID
}

func (aco AddCst) Output() WireID {
	return aco.Out
}

func (aco AddCst) Eval(cep *Protocol) {
	cep.WireOutput[aco.Out] = big.NewInt(cep.WireOutput[aco.In].Int64())
	if cep.ID == 0 {
		cep.WireOutput[aco.Out].Add(cep.WireOutput[aco.Out], big.NewInt(int64(aco.CstValue)))
	}
}

func (aco AddCst) BeaverTriplet(count int) bool {
	return false
}

type Sub struct {
	In1 WireID
	In2 WireID
	Out WireID
}

func (so Sub) Output() WireID {
	return so.Out
}

func (so Sub) Eval(cep *Protocol) {
	cep.WireOutput[so.Out] = new(big.Int).Sub(cep.WireOutput[so.In1], cep.WireOutput[so.In2])
}

func (so Sub) BeaverTriplet(count int) bool {
	return false
}

type Mult struct {
	In1 WireID
	In2 WireID
	Out WireID
}

func (mo Mult) Output() WireID {
	return mo.Out
}

// Executes a multiplication using the Beaver triplet that were already generated
func (mo Mult) Eval(cep *Protocol) {
	x := cep.WireOutput[mo.In1]
	y := cep.WireOutput[mo.In2]
	a := cep.BeaverTriplets[mo.Output()].a
	b := cep.BeaverTriplets[mo.Output()].b
	c := cep.BeaverTriplets[mo.Output()].c

	X_a := big.NewInt(0)
	X_a.Sub(x, a).Mod(X_a, q)
	Y_b := big.NewInt(0)
	Y_b.Sub(y, b).Mod(Y_b, q)

	for _, peer := range cep.Peers {
		if peer.ID != cep.ID {
			peer.Chan <- Message{MPCMessage: &MPCMessage{
				Out:   mo.Output(),
				Value: X_a.Uint64(),
			}}

			peer.Chan <- Message{MPCMessage: &MPCMessage{
				Out:   mo.Output(),
				Value: Y_b.Uint64(),
			}}
		}
	}

	for _, peer := range cep.Peers {
		if peer.ID != cep.ID {
			m := <-peer.ReceiveChan
			if m.MPCMessage == nil {
				check(errors.New("BeaverMessage received instead of MPCMessage"))
			}
			X_a.Add(X_a, big.NewInt(int64(m.MPCMessage.Value)))
			m = <-peer.ReceiveChan
			if m.MPCMessage == nil {
				check(errors.New("BeaverMessage received instead of MPCMessage"))
			}
			Y_b.Add(Y_b, big.NewInt(int64(m.MPCMessage.Value)))
		}
	}

	z := big.NewInt(0)
	z.Add(z, c)

	x_y_b := big.NewInt(0).Mul(x, Y_b)
	z.Add(z, x_y_b)

	y_x_a := big.NewInt(0).Mul(y, X_a)
	z.Add(z, y_x_a)

	if cep.ID == 0 {
		x_a_y_b := big.NewInt(0).Mul(X_a, Y_b)
		z.Sub(z, x_a_y_b)
	}

	cep.WireOutput[mo.Output()] = z
}

func (mo Mult) BeaverTriplet(count int) bool {
	return true
}

type MultCst struct {
	In       WireID
	CstValue uint64
	Out      WireID
}

func (mco MultCst) Output() WireID {
	return mco.Out
}

func (mco MultCst) Eval(cep *Protocol) {
	cep.WireOutput[mco.Out] = new(big.Int).Mul(cep.WireOutput[mco.In], big.NewInt(int64(mco.CstValue)))
}

func (mco MultCst) BeaverTriplet(count int) bool {
	return false
}

type Reveal struct {
	In  WireID
	Out WireID
}

func (ro Reveal) Output() WireID {
	return ro.Out
}

// Reveal the output by adding all the shares together
func (ro Reveal) Eval(cep *Protocol) {
	inputShare := cep.WireOutput[ro.In]
	inputShare.Mod(inputShare, q)

	for _, peer := range cep.Peers {
		if peer.ID != cep.ID {
			peer.Chan <- Message{MPCMessage: &MPCMessage{
				Out:   ro.Output(),
				Value: inputShare.Uint64(),
			}}
		}
	}

	sum := big.NewInt(0)
	sum.Add(sum, inputShare)

	for _, peer := range cep.Peers {
		if peer.ID != cep.ID {
			m := <-peer.ReceiveChan
			if m.MPCMessage == nil {
				check(errors.New("BeaverMessage received instead of MPCMessage"))
			}
			sum.Add(sum, big.NewInt(int64(m.MPCMessage.Value)))

		}
	}

	sum.Mod(sum, q)

	cep.WireOutput[ro.Output()] = sum
}

func (ro Reveal) BeaverTriplet(count int) bool {
	return false
}
