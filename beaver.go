package main

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"net"
	"sync"
)

type BeaverMessage struct {
	In1 uint64
	In2 uint64
	Out uint64
}

type BeaverTripletKey struct {
	In1 uint64
	In2 uint64
	Out uint64
}

type BeaverTripletShare struct {
	a *big.Int
	b *big.Int
	c *big.Int
}
type BeaverProtocol struct {
	sync.RWMutex
	*LocalParty
	Peers    map[PartyID]*DummyRemote
	Chan     chan DummyMessage
	Triplets map[BeaverTripletKey][]BeaverTripletShare
}

func (lp *LocalParty) NewBeaverProtocol(input uint64) *DummyProtocol {
	cep := new(DummyProtocol)
	cep.LocalParty = lp
	cep.Chan = make(chan DummyMessage, 32)
	cep.Peers = make(map[PartyID]*DummyRemote, len(lp.Peers))
	for i, rp := range lp.Peers {
		cep.Peers[i] = &DummyRemote{
			RemoteParty: rp,
			Chan:        make(chan DummyMessage, 32),
		}
	}

	cep.Input = input
	cep.Output = input
	return cep
}

func (cep *BeaverProtocol) BindNetwork(nw *TCPNetworkStruct) {
	for partyID, conn := range nw.Conns {

		if partyID == cep.ID {
			continue
		}

		rp := cep.Peers[partyID]

		// Receiving loop from remote
		go func(conn net.Conn, rp *DummyRemote) {
			for {
				var id, val uint64
				var err error
				err = binary.Read(conn, binary.BigEndian, &id)
				check(err)
				err = binary.Read(conn, binary.BigEndian, &val)
				check(err)
				msg := DummyMessage{
					Party: PartyID(id),
					Value: val,
				}
				//fmt.Println(cep, "receiving", msg, "from", rp)
				cep.Chan <- msg
			}
		}(conn, rp)
	}
}

func (cep *BeaverProtocol) Run() {

	fmt.Println(cep, "is running")

	for _, peer := range cep.Peers {
		if peer.ID != cep.ID {
			peer.Chan <- DummyMessage{cep.ID, cep.Input}
		}
	}

	received := 0
	for m := range cep.Chan {
		fmt.Println(cep, "received message from", m.Party, ":", m.Value)
		cep.Output += m.Value
		received++
		if received == len(cep.Peers)-1 {
			close(cep.Chan)
		}
	}

	if cep.WaitGroup != nil {
		cep.WaitGroup.Done()
	}
}
