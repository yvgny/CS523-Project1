package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"net"
	"sync"

	"github.com/ldsec/lattigo/ring"
)

const ThirdPartyAddr string = "localhost:1025"
const ThirdPartyID PartyID = math.MaxUint64

type BeaverMessage struct {
	PartyCount uint64
	PartyID    PartyID
	BeaverKey
	BeaverTriplet
}
type BeaverKey struct {
	In1 WireID
	In2 WireID
	Out WireID
}
type BeaverTriplet struct {
	a *big.Int
	b *big.Int
	c *big.Int
}
type BeaverProtocol struct {
	sync.RWMutex
	*LocalParty
	Peers       map[PartyID]*BeaverRemote
	Triplets    map[BeaverKey][]BeaverTriplet
	ReceiveChan chan BeaverMessage
}
type BeaverRemote struct {
	*RemoteParty
	Chan chan BeaverMessage
}

func (lp *LocalParty) NewBeaverProtocol() *BeaverProtocol {
	cep := new(BeaverProtocol)
	cep.LocalParty = lp
	cep.Peers = make(map[PartyID]*BeaverRemote, len(lp.Peers))
	cep.Triplets = make(map[BeaverKey][]BeaverTriplet)
	cep.ReceiveChan = make(chan BeaverMessage, 32)
	for i, rp := range lp.Peers {
		cep.Peers[i] = &BeaverRemote{
			RemoteParty: rp,
			Chan:        make(chan BeaverMessage, 32),
		}
	}

	return cep
}
func (cep *BeaverProtocol) BindNetwork(nw *TCPNetworkStruct) {
	for partyID, conn := range nw.Conns {

		if partyID == cep.ID {
			continue
		}

		rp := cep.Peers[partyID]

		// Receiving loop from remote
		go func(conn net.Conn, rp *BeaverRemote) {
			for {
				m := BeaverMessage{}
				var err error
				err = binary.Read(conn, binary.BigEndian, &m.PartyID)
				check(err)
				err = binary.Read(conn, binary.BigEndian, &m.PartyCount)
				check(err)
				err = binary.Read(conn, binary.BigEndian, &m.In1)
				check(err)
				err = binary.Read(conn, binary.BigEndian, &m.In2)
				check(err)
				err = binary.Read(conn, binary.BigEndian, &m.Out)
				check(err)
				//fmt.Println(cep, "receiving", m, "from", rp)
				cep.ReceiveChan <- m
			}
		}(conn, rp)

		// Sending loop of remote
		go func(conn net.Conn, rp *BeaverRemote) {
			var m BeaverMessage
			var open = true
			for open {
				m, open = <-rp.Chan
				//fmt.Println(cep, "sending", m, "to", rp)
				check(binary.Write(conn, binary.BigEndian, m.PartyID))
				check(binary.Write(conn, binary.BigEndian, m.PartyCount))
				check(binary.Write(conn, binary.BigEndian, m.In1))
				check(binary.Write(conn, binary.BigEndian, m.In2))
				check(binary.Write(conn, binary.BigEndian, m.Out))
				check(binary.Write(conn, binary.BigEndian, m.a.Uint64()))
				check(binary.Write(conn, binary.BigEndian, m.b.Uint64()))
				check(binary.Write(conn, binary.BigEndian, m.c.Uint64()))

			}
		}(conn, rp)

	}
}
func GenerateTriplets(triplets []BeaverTriplet) {
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

}
func (cep *BeaverProtocol) Run() {
	fmt.Println(cep, "is running")
	// TODO: put this in a go routine
	for m := range cep.ReceiveChan {
		key := BeaverKey{
			In1: m.In1,
			In2: m.In2,
			Out: m.Out,
		}

		if _, ok := cep.Triplets[key]; !ok {
			cep.Triplets[key] = make([]BeaverTriplet, m.PartyCount)
			GenerateTriplets(cep.Triplets[key])
		}
		triplets := cep.Triplets[key]

		m.a = triplets[m.PartyID].a
		m.b = triplets[m.PartyID].b
		m.c = triplets[m.PartyID].c
		cep.Peers[m.PartyID].Chan <- m
	}
}
