package bignum

// Int is a positive big integer of arbitrary size.
//
// Internally, an Int is stored as an array of uint16
// where the first index contains the lower 16 bits, the
// second index the next 16 bits, and so on...
//
// For example, the 64 bits integer 4611686018427387901 is
// stored as follows: [65533, 65535, 65535, 16383]
//
// The original integer can be retrieved by shifting each
// limb to the left by 16 bits * index.
//
// 65533 + (65535<<16) + (65535<<32) + (16383<<48)
// which is equivalent to
// 65533 + (65535 * 2^16) + (65535 * 2^32) + (16383 * 2^48)
type Int struct {
	nat []uint16 // natural number stored as 16 bits words
}

// NewInt initializes a big integer using an integer value
func NewInt(v int) *Int {
	bi := new(Int)
	bi.nat = storeInt(v)
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

// ToInt returns the unsigned integer representation of a big integer.
// If the big integer is larger than what an integer can contain, the
// number is truncated to fit into an integer.
func (bi *Int) ToInt() int {
	if len(bi.nat) == 0 {
		return 0
	}
	v := int(bi.nat[0])
	if len(bi.nat) == 1 {
		return v
	}
	for i := 1; i < 4 && i < len(bi.nat); i++ {
		v = int(bi.nat[i])<<(16*i) | v
	}
	return v
}

// SetBytes sets the value of a big integer to the provided byte buffer.
//
// The buffer must contain a big-endian unsigned integer. For example,
// the integer 3545084735 would be written as 0xD34DB33F and its []byte
// would be []byte{0xD3, 0x4D, 0xB3, 0x3F}.
//
// When stored in the Int nat slice, the order of the bytes is reversed,
// such that the lower 16 bits of the number, the last two bytes of the buf
// slice, is stored in the first index of the Int nat slice. And the upper
// 16 bits of the number are stored in the last index entry of the Int nat slice.
//
// If the provided buf is of an odd length, then the last uint16 is only 8 bits
// long. It is still stored as a uint16, with the upper 8 bits set to zero.
func (bi *Int) SetBytes(buf []byte) {
	bi.nat = bi.nat[:0]
	for i := len(buf) - 1; i >= 0; i -= 2 {
		// bound check if we're at the last byte of an odd slice.
		// if so, the upper 8 bits of the last limbs are set to zero
		if i == 0 {
			bi.nat = append(bi.nat, uint16(buf[i]))
			break
		}
		// convert two bytes into a uint16 and append them
		// to the nat slice
		bi.nat = append(bi.nat, uint16(buf[i-1])<<8|uint16(buf[i]))
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
		limb16 >>= 8
		buf = append([]byte{byte(limb16)}, buf...)
		i += 8
		if i >= len(bi.nat)*16 {
			break
		}
	}
	// strip leading zeroes
	for i = 0; buf[i] == 0 && i < len(buf); i++ {
	}
	return buf[i:]
}

// Add provides addition on big integers. It takes an *Int
// as argument and adds its value to the bi.
//
// This algorithm isn't particularly smart. It simply adds
// each uint16 from bi and x at the same index to each other
// into an uint32, then keep the upper 16 bits as a carry for
// the next index, and store the lower 16 bits at the index.
//
// If the carry is not zero after the last addition, it is
// appended to the nat slice of bi.
func (bi *Int) Add(x *Int) {
	switch {
	case len(bi.nat) < len(x.nat):
		x.Add(bi)
		*bi = *x
		return
	case len(x.nat) == 0:
		return
	case len(bi.nat) == 0:
		*bi = *x
		return
	}
	carry := uint32(0)
	// add all limbs from x, the smallest number, to bi
	for i := 0; i < len(x.nat); i++ {
		limbsum32 := uint32(bi.nat[i]) + uint32(x.nat[i]) + carry
		//fmt.Printf("%x + %x + %x = %x; carry=%x; bitlen=%x\n", bi.nat[i], x.nat[i], carry, limbsum32)
		carry = uint32(limbsum32 >> 16)
		//fmt.Printf("carry=%x;\n", carry)
		bi.nat[i] = uint16(limbsum32 & 0xFFFF)
	}
	// if there's a remaining carry, either add it to an upper limb of bi
	// or allocate a new limb if needed
	if carry == 1 {
		if len(bi.nat) == len(x.nat) {
			bi.nat = append(bi.nat, uint16(1))
		} else {
			bi.nat[len(x.nat)] = uint16(bi.nat[len(x.nat)] + 1)
		}
	}
}

// Mul implements multiplication of the provided Int x with bi
//
// It uses a naive linear convolution algorithm that multiplies
// uint16 words one by one, starting with the lower ones at the
// beginning of the nat slices.
func (bi *Int) Mul(x *Int) {
	switch {
	case len(bi.nat) < len(x.nat):
		x.Mul(bi)
		*bi = *x
		return
	case len(x.nat) == 0, len(bi.nat) == 0:
		// multiplication by zero just sets bi to zero
		bi.nat[0] = 0
		return
	}

	var product = new(Int)
	for i := 0; i < len(bi.nat); i++ {
		var inter = new(Int)
		for j := 0; j < len(x.nat); j++ {
			p := NewInt(int(bi.nat[i]) * int(x.nat[j]))
			// raise p by 2^16 for each word already processed
			p.shift16(j)
			//fmt.Printf("p=%d*%d=%+v\n", bi.nat[i], x.nat[j], p)
			inter.Add(p)
			//fmt.Printf("inter=%+v\n", inter)
		}
		// raise inter by 2^16 for each word already processed
		inter.shift16(i)
		//fmt.Printf("inter=%+v\n", inter)
		product.Add(inter)
	}
	*bi = *product
	//fmt.Printf("bi=%+v\n", bi)
}

// shift bi by x count of 16 bits words
func (bi *Int) shift16(count int) {
	for shift := 0; shift < count; shift++ {
		bi.nat = append([]uint16{0}, bi.nat...)
	}
}
