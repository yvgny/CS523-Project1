package main

import (
	"github.com/ldsec/lattigo/ring"
)

func newRandomVec(n, T uint64) []uint64 {
	vec := make([]uint64, n)
	t := ring.NewUint(T)
	for i := range vec {
		vec[i] = ring.RandInt(t).Uint64()
	}

	return vec
}

func addVec(a, b []uint64, T uint64) []uint64 {
	res := make([]uint64, len(a))
	t := ring.NewUint(T)
	for i := range res {
		add := ring.NewUint(0)
		add.Add(ring.NewUint(a[i]), ring.NewUint(b[i])).Mod(add, t)
		res[i] = add.Uint64()

	}

	return res
}

func subVec(a, b []uint64, T uint64) []uint64 {
	res := make([]uint64, len(a))
	t := ring.NewUint(T)
	for i := range res {
		sub := ring.NewUint(0)
		sub.Sub(ring.NewUint(a[i]), ring.NewUint(b[i])).Mod(sub, t)
		res[i] = sub.Uint64()

	}

	return res
}

func mulVec(a, b []uint64, T uint64) []uint64 {
	res := make([]uint64, len(a))
	t := ring.NewUint(T)
	for i := range res {
		mul := ring.NewUint(0)
		mul.Mul(ring.NewUint(a[i]), ring.NewUint(b[i])).Mod(mul, t)
		res[i] = mul.Uint64()

	}

	return res
}

func negVec(a []uint64, T uint64) []uint64 {
	res := make([]uint64, len(a))
	t := ring.NewUint(T)
	for i := range res {
		neg := ring.NewUint(a[i])
		neg.Neg(neg).Mod(neg, t)
		res[i] = neg.Uint64()

	}

	return res
}
