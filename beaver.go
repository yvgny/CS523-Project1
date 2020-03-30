package main

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/ldsec/lattigo/bfv"
)

type BeaverMessage struct {
	Size  int
	Value []byte
}

type BeaverProtocol struct {
	*LocalParty
	Params         *bfv.Parameters
	Peers          map[PartyID]*BeaverRemoteParty
	Encoder        bfv.Encoder
	Evaluator bfv.Evaluator
	BeaverTriplets Triplet
}

type Triplet struct {
	b *bfv.Plaintext
	c []uint64
	sk *bfv.SecretKey
}

type BeaverRemoteParty struct {
	*RemoteParty
	Chan        chan BeaverMessage
	ReceiveChan chan BeaverMessage
}

func (lp *LocalParty) NewBeaverProtocol(params *bfv.Parameters) *BeaverProtocol {
	cep := new(BeaverProtocol)
	cep.LocalParty = lp
	cep.Peers = make(map[PartyID]*BeaverRemoteParty, len(lp.Peers))
	cep.Params = params
	cep.Encoder = bfv.NewEncoder(params)
	cep.Evaluator = bfv.NewEvaluator(params)
	for i, rp := range lp.Peers {
		cep.Peers[i] = &BeaverRemoteParty{
			RemoteParty: rp,
			Chan:        make(chan BeaverMessage, 32),
			ReceiveChan: make(chan BeaverMessage, 32),
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
		go func(conn net.Conn, rp *BeaverRemoteParty) {
			for {
				var size int

				var err error
				err = binary.Read(conn, binary.BigEndian, &size)
				check(err)
				val := make([]byte, size)
				err = binary.Read(conn, binary.BigEndian, &val)
				check(err)

				msg := BeaverMessage{Size:size, Value:val}
				//fmt.Println(cep, "receiving", msg, "from", rp)
				rp.ReceiveChan <- msg
			}
		}(conn, rp)

		// Sending loop of remote
		go func(conn net.Conn, rp *BeaverRemoteParty) {
			var m BeaverMessage
			var open = true
			for open {
				m, open = <-rp.Chan
				//fmt.Println(cep, "sending", m, "to", rp)
				check(binary.Write(conn, binary.BigEndian, m.Size))
				check(binary.Write(conn, binary.BigEndian, m.Value))
			}
		}(conn, rp)
	}
}

func (cep *BeaverProtocol) Run() {

	fmt.Println(cep, "is running")

	cep.GenerateTriplets()
	cep.ReceiveOtherBeaver()
	cep.ComputeC()

	if cep.WaitGroup != nil {
		cep.WaitGroup.Done()
	}

}

func (cep *BeaverProtocol) ComputeC() {
	encC := bfv.NewCiphertext(cep.Params, 1)
	for id, peer := range cep.Peers {
		if id != cep.ID {
			msg := <-peer.ReceiveChan
			dij := bfv.NewCiphertext(cep.Params, 1)
			err := dij.UnmarshalBinary(msg.Value)
			check(err)
			cep.Evaluator.Add(encC, dij, encC)
		}
	}

	decryptor := bfv.NewDecryptor(cep.Params, cep.BeaverTriplets.sk)
	decC := decryptor.DecryptNew(encC)
	decCVec := cep.Encoder.DecodeUint(decC)
	cep.BeaverTriplets.c = addVec(cep.BeaverTriplets.c, decCVec, cep.Params.T)
}

func (cep *BeaverProtocol) ReceiveOtherBeaver() {
	bi := cep.BeaverTriplets.b
	for id, peer := range cep.Peers {
		if id != cep.ID {
			msg := <-peer.ReceiveChan
			dj := bfv.NewCiphertext(cep.Params, 1)
			err := dj.UnmarshalBinary(msg.Value)
			check(err)

			rij := newRandomVec(1<<cep.Params.LogN, cep.Params.T)

			cep.BeaverTriplets.c = subVec(cep.BeaverTriplets.c, rij, cep.Params.T)

			rijPt := bfv.NewPlaintext(cep.Params)
			cep.Encoder.EncodeUint(rij, rijPt)

			mul := bfv.NewCiphertext(cep.Params, 1)
			cep.Evaluator.Mul(dj, bi, mul)

			//TODO: errors (ligne 19 et 20 protocol)
			dij := bfv.NewCiphertext(cep.Params, 1)
			cep.Evaluator.Add(mul, rijPt, dij)

			bytes, err := dij.MarshalBinary()
			check(err)
			beaverMessage := BeaverMessage{Size: len(bytes), Value: bytes}
			peer.Chan <- beaverMessage

		}
	}
}

func (cep *BeaverProtocol) GenerateTriplets() {
	keyGen := bfv.NewKeyGenerator(cep.Params)

	ai := newRandomVec(1<<cep.Params.LogN, cep.Params.T)
	bi := newRandomVec(1<<cep.Params.LogN, cep.Params.T)
	ci := mulVec(ai, bi, cep.Params.T)

	aiPt := bfv.NewPlaintext(cep.Params)
	cep.Encoder.EncodeUint(ai, aiPt)

	biPt := bfv.NewPlaintext(cep.Params)
	cep.Encoder.EncodeUint(bi, biPt)

	ski := keyGen.GenSecretKey()

	cep.BeaverTriplets = Triplet{b: biPt, c: ci, sk:ski}

	encryptor := bfv.NewEncryptorFromSk(cep.Params, ski)
	di := encryptor.EncryptNew(aiPt)
	bytes, err := di.MarshalBinary()
	check(err)
	beaverMessage := BeaverMessage{Size: len(bytes), Value: bytes}

	for id, peer := range cep.Peers {
		if id != cep.ID {
			peer.Chan <- beaverMessage
		}
	}


}