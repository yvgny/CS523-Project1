package main

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"net"

	"github.com/ldsec/lattigo/ring"
)

var q = big.NewInt(1<<16 + 1)

type Message struct {
	Party PartyID
	Value uint64
}

type Protocol struct {
	*LocalParty
	Chan            chan Message
	Peers           map[PartyID]*Remote
	ThirdPartyChans ThirdPartyChannels

	Input      uint64
	Output     uint64
	Circuit    Circuit
	InputShare *big.Int
}

type Remote struct {
	*RemoteParty
	Chan chan Message
}

type ThirdPartyChannels struct {
	Receive chan BeaverMessage
	Send    chan BeaverMessage
}

func (lp *LocalParty) NewProtocol(input uint64, circuit Circuit) *Protocol {
	cep := new(Protocol)
	cep.LocalParty = lp
	cep.Chan = make(chan Message, 32)
	cep.Peers = make(map[PartyID]*Remote, len(lp.Peers))
	cep.Circuit = circuit

	delete(lp.Peers, ThirdPartyID)
	for i, rp := range lp.Peers {
		cep.Peers[i] = &Remote{
			RemoteParty: rp,
			Chan:        make(chan Message, 32),
		}
	}

	cep.ThirdPartyChans = ThirdPartyChannels{
		Receive: make(chan BeaverMessage),
		Send:    make(chan BeaverMessage),
	}

	cep.Input = input
	return cep
}

func (cep *Protocol) BindNetwork(nw *TCPNetworkStruct) {
	for partyID, conn := range nw.Conns {

		if partyID == cep.ID || partyID == ThirdPartyID {
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

	// Receiving loop from remote
	go func(conn net.Conn) {
		for {
			m := BeaverMessage{}
			var a, b, c uint64
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
			cep.ThirdPartyChans.Receive <- m
		}
	}(nw.Conns[ThirdPartyID])

	// Sending loop of remote
	go func(conn net.Conn) {
		var m BeaverMessage
		var open = true
		for open {
			m, open = <-cep.ThirdPartyChans.Send
			//fmt.Println(cep, "sending", m, "to", rp)
			check(binary.Write(conn, binary.BigEndian, m.PartyID))
			check(binary.Write(conn, binary.BigEndian, m.PartyCount))
			check(binary.Write(conn, binary.BigEndian, m.In1))
			check(binary.Write(conn, binary.BigEndian, m.In2))
			check(binary.Write(conn, binary.BigEndian, m.Out))
		}
	}(nw.Conns[ThirdPartyID])
}

func (cep *Protocol) Run() {

	fmt.Println(cep, "is running")

	sum := big.NewInt(0)
	for _, peer := range cep.Peers {
		if peer.ID != cep.ID {

			share := ring.RandInt(q)

			sum.Add(sum, share)
			peer.Chan <- Message{cep.ID, share.Uint64()}
		}
	}
	s := big.NewInt(int64(cep.Input))
	cep.InputShare = new(big.Int).Sub(s, sum)
	cep.InputShare.Mod(cep.InputShare, q)

	inputShares := make(map[PartyID]*big.Int)
	resultShares := make(map[PartyID]*big.Int)
	inputShares[cep.ID] = cep.InputShare
	received := 0
	for m := range cep.Chan {
		if _, ok := inputShares[m.Party]; ok {
			resultShares[m.Party] = big.NewInt(int64(m.Value))
			break
		}
		fmt.Println(cep, "received input share from", m.Party, ":", m.Value)
		received++
		inputShares[m.Party] = big.NewInt(int64(m.Value))
		if received == len(cep.Peers)-1 {
			break
		}
	}

	resShare, err := Eval(cep, cep.Circuit, inputShares, cep.ID)
	check(err)

	for _, peer := range cep.Peers {
		if peer.ID != cep.ID {
			peer.Chan <- Message{cep.ID, resShare.Uint64()}
		}
	}

	received = 0
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
