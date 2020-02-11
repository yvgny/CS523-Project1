package main

type WireID uint64

type GateID uint64

type Operation interface {
	Output() WireID
}

type Input struct {
	Party PartyID
	Out   WireID
}

func (io Input) Output() WireID {
	return io.Out
}

type Add struct {
	In1 WireID
	In2 WireID
	Out WireID
}

func (ao Add) Output() WireID {
	return ao.Out
}

type AddCst struct {
	In WireID
	CstValue uint64
	Out WireID
}

func (aco AddCst) Output() WireID {
	return aco.Out
}

type Sub struct {
	In1 WireID
	In2 WireID
	Out WireID
}

func (so Sub) Output() WireID {
	return so.Out
}

type Mult struct {
	In1 WireID
	In2 WireID
	Out WireID
}

func (mo Mult) Output() WireID {
	return mo.Out
}

type MultCst struct {
	In WireID
	CstValue uint64
	Out WireID
}

func (mco MultCst) Output() WireID {
	return mco.Out
}

type Reveal struct {
	In WireID
	Out WireID
}

func (ro Reveal) Output() WireID {
	return ro.Out
}
