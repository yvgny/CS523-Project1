package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"
	"net"
)

var q = big.NewInt(1<<16 + 1)

type Message struct {
	Party PartyID
	Value uint64
}

type Protocol struct {
	*LocalParty
	Chan  chan Message
	Peers map[PartyID]*Remote

	Input      uint64
	Output     uint64
	Circuit    Circuit
	InputShare *big.Int
}

type Remote struct {
	*RemoteParty
	Chan chan Message
}

func (lp *LocalParty) NewProtocol(input uint64, circuit Circuit) *Protocol {
	cep := new(Protocol)
	cep.LocalParty = lp
	cep.Chan = make(chan Message, 32)
	cep.Peers = make(map[PartyID]*Remote, len(lp.Peers))
	cep.Circuit = circuit
	for i, rp := range lp.Peers {
		cep.Peers[i] = &Remote{
			RemoteParty: rp,
			Chan:        make(chan Message, 32),
		}
	}

	cep.Input = input
	return cep
}

func (cep *Protocol) BindNetwork(nw *TCPNetworkStruct) {
	for partyID, conn := range nw.Conns {

		if partyID == cep.ID {
			continue
		}

		rp := cep.Peers[partyID]

		// Receiving loop from remote
		go func(conn net.Conn, rp *Remote) {
			for {
				var id, val uint64
				var err error
				err = binary.Read(conn, binary.BigEndian, &id)
				check(err)
				err = binary.Read(conn, binary.BigEndian, &val)
				check(err)
				msg := Message{
					Party: PartyID(id),
					Value: val,
				}
				//fmt.Println(cep, "receiving", msg, "from", rp)
				cep.Chan <- msg
			}
		}(conn, rp)

		// Sending loop of remote
		go func(conn net.Conn, rp *Remote) {
			var m Message
			var open = true
			for open {
				m, open = <-rp.Chan
				//fmt.Println(cep, "sending", m, "to", rp)
				check(binary.Write(conn, binary.BigEndian, m.Party))
				check(binary.Write(conn, binary.BigEndian, m.Value))
			}
		}(conn, rp)
	}
}

func (cep *Protocol) Run() {

	fmt.Println(cep, "is running")

	sum := big.NewInt(0)
	for _, peer := range cep.Peers {
		if peer.ID != cep.ID {

			share, err := rand.Int(rand.Reader, q)
			check(err)

			sum.Add(sum, share)
			peer.Chan <- Message{cep.ID, share.Uint64()}
		}
	}
	s := big.NewInt(int64(cep.Input))
	cep.InputShare = new(big.Int).Sub(s, sum)
	cep.InputShare.Mod(cep.InputShare, q)

	inputShares := make(map[PartyID]*big.Int)
	received := 0
	for m := range cep.Chan {
		fmt.Println(cep, "received input share from", m.Party, ":", m.Value)
		received++
		inputShares[m.Party] = big.NewInt(int64(m.Value))
		if received == len(cep.Peers)-1 {
			break
		}
	}

	resShare, err := Eval(cep.Circuit, inputShares, cep.ID)
	check(err)

	for _, peer := range cep.Peers {
		if peer.ID != cep.ID {
			peer.Chan <- Message{cep.ID, resShare.Uint64()}
		}
	}

	received = 0
	resultShares := make(map[PartyID]*big.Int)
	resultShares[cep.ID] = resShare
	for m := range cep.Chan {
		fmt.Println(cep, "received result share from", m.Party, ":", m.Value)
		received++
		resultShares[m.Party] = big.NewInt(int64(m.Value))
		if received == len(cep.Peers)-1 {
			close(cep.Chan)
		}
	}

	reveal := big.NewInt(0)
	for _, s := range resultShares {
		reveal.Add(reveal, s)
	}

	reveal.Mod(reveal, q)

	if cep.WaitGroup != nil {
		cep.WaitGroup.Done()
	}

	cep.Output = reveal.Uint64()
}
