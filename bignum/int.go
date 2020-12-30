package bignum

// Int is a positive big integer
type Int struct {
	nat    []uint16 // natural number stored as 16 bits words
	natlen int      // length of the nat slice
	bitlen int      // bit length of the integer
}

// NewInt initializes a big integer using a small uint16 value
func NewInt(v int) *Int {
	bi := new(Int)
	bi.nat = storeInt(v)
	bi.natlen = len(bi.nat)
	bi.bitlen = bi.natlen * 16
	return bi
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

// ToInt returns the unsigned integer representation of a big integer
func (bi *Int) ToInt() int {
	if bi.natlen == 0 {
		return 0
	}
	v := int(bi.nat[0])
	if bi.natlen == 1 {
		return v
	}
	for i := 1; i < 4 && i < bi.natlen; i++ {
		v = int(bi.nat[i])<<(16*i) | v
	}
	return v
}

// SetBytes sets the value of the big integer to the provided byte buffer.
// It assumes the buffer contains a big-endian unsigned integer.
func (bi *Int) SetBytes(buf []byte) {
	bi.nat = bi.nat[:0]
	for i := len(buf) - 1; i >= 0; i -= 2 {
		// bound check if we're at the last byte of an odd slice.
		// if so, the upper 8 bits of the last limbs are set to zero
		// and the bitlen is incremented by 8 instead of 16
		if i == 0 {
			bi.nat = append(bi.nat, uint16(buf[i]))
			bi.natlen = len(bi.nat)
			bi.bitlen += 8
			break
		}
		// convert two bytes into a uint16 and append them
		// to the nat slice
		bi.nat = append(bi.nat, uint16(buf[i-1])<<8|uint16(buf[i]))
		bi.natlen = len(bi.nat)
		bi.bitlen += 16
	}
}

// Bytes returns the big endian unsigned byte slice representation
// of the big integer
func (bi *Int) Bytes() []byte {
	i := 0
	var buf []byte
	for _, limb16 := range bi.nat {
		buf = append([]byte{byte(limb16)}, buf...)
		i += 8
		if i >= bi.bitlen {
			break
		}
		limb16 >>= 8
		buf = append([]byte{byte(limb16)}, buf...)
		i += 8
		if i >= bi.bitlen {
			break
		}
	}
	// strip leading zeroes
	for i = 0; buf[i] == 0 && i < len(buf); i++ {
	}
	return buf[i:]
}

// Add provides addition on big integers
func (bi *Int) Add(x *Int) {
	bitlen := 0
	switch {
	case bi.natlen < x.natlen:
		x.Add(bi)
		bi = x
		return
	case x.natlen == 0:
		return
	case bi.natlen == 0:
		bi = x
		return
	}
	// bi.natlen >= x.natlen, continue here
	carry := uint32(0)
	// add all limbs from x, the smallest number, to bi
	for i := 0; i < x.natlen; i++ {
		limbsum32 := uint32(bi.nat[i]) + uint32(x.nat[i]) + carry
		//fmt.Printf("%x + %x + %x = %x; ", bi.nat[i], x.nat[i], carry, limbsum32)
		carry = uint32(limbsum32 >> 16)
		//fmt.Printf("carry=%x\n", carry)
		bi.nat[i] = uint16(limbsum32 & 0xFFFF)
		bitlen += 16
	}
	// if there's a remaining carry, either add it to an upper limb of bi
	// or allocate a new limb if needed
	if carry == 1 {
		if bi.natlen == x.natlen {
			bi.nat = append(bi.nat, uint16(1))
			// bi grew by one, update its length
			bi.natlen++
			bitlen += 8
		} else {
			bi.nat[x.natlen] = uint16(bi.nat[x.natlen] + 1)
		}
	}
	bi.bitlen = bitlen
}
