package main

import (
	"encoding/binary"
	"fmt"
	"github.com/ldsec/lattigo/ring"
	"math/big"
	"net"
	"sync"
)

const ThirdPartyAddr string = "localhost:1025"


type BeaverMessage struct {
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
	Triplets []BeaverMessage
}
type BeaverRemote struct {
	*RemoteParty
	Chan chan BeaverMessage
}

func (lp *LocalParty) NewBeaverProtocol(circuit []Operation) *BeaverProtocol {
	cep := new(BeaverProtocol)
	cep.LocalParty = lp
	cep.Peers = make(map[PartyID]*BeaverRemote, len(lp.Peers))
	cep.Triplets = make([]BeaverMessage, 0)
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
				var a,b,c uint64
				var err error
				err = binary.Read(conn, binary.BigEndian, &m.In1)
				check(err)
				err = binary.Read(conn, binary.BigEndian, &m.In2)
				check(err)
				err = binary.Read(conn, binary.BigEndian, &m.Out)
				check(err)
				err = binary.Read(conn, binary.BigEndian, &a)
				check(err)
				m.a = big.NewInt(int64(a))
				err = binary.Read(conn, binary.BigEndian, &b)
				check(err)
				m.b = big.NewInt(int64(b))
				err = binary.Read(conn, binary.BigEndian, &c)
				check(err)
				m.c = big.NewInt(int64(c))
				//fmt.Println(cep, "receiving", msg, "from", rp)
				cep.Chan <- msg
			}
		}(conn, rp)

		// Sending loop of remote
		go func(conn net.Conn, rp *BeaverRemote) {
			var m BeaverMessage
			var open = true
			for open {
				m, open = <-rp.Chan
				//fmt.Println(cep, "sending", m, "to", rp)
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
				peer.Chan <- BeaverMessage{&share}

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
				peer.Chan <- BeaverMessage{&share}
			}
			i++
		}
	}

	for _, peer := range cep.Peers{
		peer.Chan <- BeaverMessage{}
	}

}
