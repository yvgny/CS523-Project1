package main

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ldsec/lattigo/ring"
)

type BeaverMessage struct {
	Triplet BeaverTriplet
}

type BeaverTriplet struct {
	In1 WireID
	In2 WireID
	Out WireID
	a   *big.Int
	b   *big.Int
	c   *big.Int
}
type BeaverProtocol struct {
	sync.RWMutex
	*LocalParty
	Peers    map[PartyID]*BeaverRemote
	Triplets []BeaverTriplet
}
type BeaverRemote struct {
	*RemoteParty
	Chan chan BeaverMessage
}

func (lp *LocalParty) NewBeaverProtocol(circuit []Operation) *BeaverProtocol {
	cep := new(BeaverProtocol)
	cep.LocalParty = lp
	cep.Peers = make(map[PartyID]*BeaverRemote, len(lp.Peers))
	cep.Triplets = make([]BeaverTriplet, 0)
	for _, op := range circuit {
		triplet := op.GenerateTriplet(len(cep.Peers))
		if triplet != nil {
			cep.Triplets = append(cep.Triplets, *triplet)
		}

	}
	for i, rp := range lp.Peers {
		cep.Peers[i] = &BeaverRemote{
			RemoteParty: rp,
			Chan:        make(chan BeaverMessage, 32),
		}
	}

	return cep
}

func (cep *BeaverProtocol) Run() {

	fmt.Println(cep, "is running")

	for _, triplet := range cep.Triplets {
		sum_a := big.NewInt(0)
		sum_b := big.NewInt(0)
		sum_c := big.NewInt(0)
		i := 0
		for _, peer := range cep.Peers {
			if i != len(cep.Peers)-1 {
				a_share := ring.RandInt(q)
				b_share := ring.RandInt(q)
				c_share := ring.RandInt(q)
				sum_a.Add(a_share, sum_a)
				sum_b.Add(b_share, sum_b)
				sum_c.Add(c_share, sum_c)
				share := BeaverTriplet{
					In1: triplet.In1,
					In2: triplet.In2,
					Out: triplet.Out,
					a:   a_share,
					b:   b_share,
					c:   c_share,
				}
				peer.Chan <- BeaverMessage{share}

			} else {
				a_share := big.NewInt(0)
				a_share.Sub(triplet.a, sum_a).Mod(a_share, q)

				b_share := big.NewInt(0)
				b_share.Sub(triplet.b, sum_b).Mod(b_share, q)

				c_share := big.NewInt(0)
				c_share.Sub(triplet.c, sum_c).Mod(c_share, q)

				share := BeaverTriplet{
					In1: triplet.In1,
					In2: triplet.In2,
					Out: triplet.Out,
					a:   a_share,
					b:   b_share,
					c:   c_share,
				}
				peer.Chan <- BeaverMessage{share}
			}
			i++
		}
	}

}
