package main

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/ldsec/lattigo/ring"
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
	Eval(*Protocol)
	BeaverTriplet(int) []BeaverTriplet
}

func (io Input) generateShares(cep *Protocol) {
	sum := big.NewInt(0)
	for _, peer := range cep.Peers {
		if peer.ID != cep.ID {

			share, err := rand.Int(rand.Reader, q)
			check(err)

			sum.Add(sum, share)
			peer.Chan <- MPCMessage{io.Out, share.Uint64()}
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

func (io Input) Eval(cep *Protocol) {
	if io.Party == cep.ID {
		io.generateShares(cep)
	} else {
		m := <-cep.Peers[io.Party].ReceiveChan
		cep.WireOutput[io.Out] = big.NewInt(int64(m.Value))
	}
}

func (io Input) BeaverTriplet(count int) []BeaverTriplet {
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

func (ao Add) Eval(cep *Protocol) {
	cep.WireOutput[ao.Out] = new(big.Int).Add(cep.WireOutput[ao.In1], cep.WireOutput[ao.In2])
}

func (ao Add) BeaverTriplet(count int) []BeaverTriplet {
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

func (aco AddCst) Eval(cep *Protocol) {
	cep.WireOutput[aco.Out] = big.NewInt(cep.WireOutput[aco.In].Int64())
	if cep.ID == 0 {
		cep.WireOutput[aco.Out].Add(cep.WireOutput[aco.Out], big.NewInt(int64(aco.CstValue)))
	}
}

func (aco AddCst) BeaverTriplet(count int) []BeaverTriplet {
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

func (so Sub) Eval(cep *Protocol) {
	cep.WireOutput[so.Out] = new(big.Int).Sub(cep.WireOutput[so.In1], cep.WireOutput[so.In2])
}

func (so Sub) BeaverTriplet(count int) []BeaverTriplet {
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
			peer.Chan <- MPCMessage{
				Out:   mo.Output(),
				Value: X_a.Uint64(),
			}

			peer.Chan <- MPCMessage{
				Out:   mo.Output(),
				Value: Y_b.Uint64(),
			}
		}
	}

	for _, peer := range cep.Peers {
		if peer.ID != cep.ID {
			m := <-peer.ReceiveChan
			X_a.Add(X_a, big.NewInt(int64(m.Value)))
			m = <-peer.ReceiveChan
			Y_b.Add(Y_b, big.NewInt(int64(m.Value)))
		}
	}

	fmt.Println("id:", cep.ID, "x,y: ", x.Uint64(), y.Uint64(), "a, b, c", a.Uint64(), b.Uint64(), c.Uint64(), "x-a, y-b:", X_a.Uint64(), Y_b.Uint64())

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

func (mo Mult) BeaverTriplet(count int) []BeaverTriplet {
	triplets := make([]BeaverTriplet, count)

	sum_a := big.NewInt(0)
	sum_b := big.NewInt(0)
	sum_c := big.NewInt(0)
	a := ring.RandInt(q)
	b := ring.RandInt(q)
	c := big.NewInt(0).Mul(a, b)
	c.Mod(c, q)
	for i := 0; i < len(triplets)-1; i++ {
		a_share := ring.RandInt(q)
		b_share := ring.RandInt(q)
		c_share := ring.RandInt(q)
		sum_a.Add(a_share, sum_a)
		sum_b.Add(b_share, sum_b)
		sum_c.Add(c_share, sum_c)
		triplets[i] = BeaverTriplet{
			a: a_share,
			b: b_share,
			c: c_share,
		}
	}

	a_share := big.NewInt(0)
	a_share.Sub(a, sum_a).Mod(a_share, q)

	b_share := big.NewInt(0)
	b_share.Sub(b, sum_b).Mod(b_share, q)

	c_share := big.NewInt(0)
	c_share.Sub(c, sum_c).Mod(c_share, q)

	triplets[len(triplets)-1] = BeaverTriplet{
		a: a_share,
		b: b_share,
		c: c_share,
	}

	return triplets
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

func (mco MultCst) BeaverTriplet(count int) []BeaverTriplet {
	return nil
}

type Reveal struct {
	In  WireID
	Out WireID
}

func (ro Reveal) Output() WireID {
	return ro.Out
}

func (ro Reveal) Eval(cep *Protocol) {
	inputShare := cep.WireOutput[ro.In]
	inputShare.Mod(inputShare, q)

	for _, peer := range cep.Peers {
		if peer.ID != cep.ID {
			peer.Chan <- MPCMessage{
				Out:   ro.Output(),
				Value: inputShare.Uint64(),
			}

		}
	}

	sum := big.NewInt(0)
	sum.Add(sum, inputShare)

	for _, peer := range cep.Peers {
		if peer.ID != cep.ID {
			m := <-peer.ReceiveChan
			sum.Add(sum, big.NewInt(int64(m.Value)))

		}
	}

	sum.Mod(sum, q)

	cep.WireOutput[ro.Output()] = sum
}

func (ro Reveal) BeaverTriplet(count int) []BeaverTriplet {
	return nil
}
