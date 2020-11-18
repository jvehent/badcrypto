package bignum

import "fmt"

// Int is a bit integer
type Int struct {
	neg    bool     // sign
	nat    []uint16 // natural number stored as 16 bits words
	natlen int      // length of the nat slice
}

// NewInt initializes a big integer using a small uint16 value
func NewInt(v int) *Int {
	i := new(Int)
	if v < 0 {
		i.neg = true
		v = -v
	}
	i.nat = storeInt(v)
	i.natlen = len(i.nat)
	return i
}

func storeInt(v int) []uint16 {
	nat := make([]uint16, 0)
	if v == 0 {
		nat = append(nat, uint16(0))
		return nat
	}
	for i := v; i > 0; i = i >> 16 {
		limb := uint16(i & 0xFFFF)
		nat = append(nat, limb)
	}
	return nat
}

func uInt16FromBigEndianWord(b []byte) uint16 {
	if len(b) != 2 {
		return uint16(0)
	}
	return uint16(b[1]) | uint16(b[0])<<8

}

// ToInt returns the unsigned integer representation of a big integer
func (bi *Int) ToInt() int {
	v := int(bi.nat[0])
	fmt.Printf("%+v\n", bi.nat)
	switch bi.natlen {
	case 0:
		return 0
	case 1:
		// do nothing else
	default:
		for i := 1; i < 4 && i < bi.natlen; i++ {
			v = int(bi.nat[i])<<(16*i) | v
		}
	}
	// set the sign
	if bi.neg {
		v = -v
	}
	return v
}
