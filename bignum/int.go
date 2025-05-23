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
	bi.nat = bi.nat[:0] // clear the buffer
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

// Set sets bi to the value of x
func (bi *Int) Set(x *Int) {
	bi.nat = make([]uint16, len(x.nat))
	copy(bi.nat, x.nat)
}

// Bytes returns the big endian unsigned byte slice representation
// of the big integer
func (bi *Int) Bytes() []byte {
	i := 0
	if len(bi.nat) == 0 {
		return []byte{}
	}
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
	for i = 0; buf[i] == 0 && i < len(buf)-1; i++ {
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
		//fmt.Printf("%x + %x + %x = %x\n", bi.nat[i], x.nat[i], carry, limbsum32)
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

// Sub substracts x from bi. If x is greater than bi, it panics.
func (bi *Int) Sub(x *Int) {
	switch bi.Compare(x) {
	case -1:
		panic("x is larger than bi, which would result in a negative number, that are not yet supported")
	case 0:
		bi.Zero()
		return
	}
	carry := int(0)
	var i int
	for i = 0; i < len(x.nat); i++ {
		limbdiff32 := int(bi.nat[i]) - (int(x.nat[i]) + carry)
		//fmt.Printf("%x - (%x + %x) = %x;\n", bi.nat[i], x.nat[i], carry, limbdiff32)
		if limbdiff32 < 0 {
			// x.nat[i] was greater than bi.nat[i] so the diff is a negative
			// number. we store a carry of one and set the value of bi.nat[i]
			// to the inverse of the difference
			carry = 1
			bi.nat[i] = 0xFFFF - uint16(-limbdiff32) + 1 // surely there's a better way...
			//fmt.Printf("storing bi.nat[%x]=%x\n", i, 0xFFFF-uint16(-limbdiff32))
		} else {
			carry = 0
			bi.nat[i] = uint16(limbdiff32)
		}
	}
	if carry == 1 {
		if len(bi.nat) == len(x.nat) {
			panic("remaining carry implies x is larger than bi and negative numbers are not supported")
		}
		//fmt.Printf("i=%d; len(bi.nat)=%d; len(x.nat)=%d\n", i, len(bi.nat), len(x.nat))
		bi.nat[i]--
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
		y := new(Int)
		y.Set(x)
		y.Mul(bi)
		*bi = *y
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

// Div implements integer division of bi by x and returns
// the remainder n.
func (bi *Int) Div(x *Int) (n *Int) {
	// Ensure inputs are normalized, especially for x if it's [0]
	// and for bi for iteration.
	bi.normalize()
	x.normalize()

	// n is a named return parameter, initialized to nil.
	// It must be set before any return.

	if len(x.nat) == 1 && x.nat[0] == 0 { // x is zero
		panic("division by zero")
	}

	// Special case: If bi (dividend) is zero.
	// Use Compare with NewInt(0) for robustness.
	// NewInt(0) creates {nat: [0]}, which is our canonical zero.
	tempZeroForCompare := NewInt(0) 
	if bi.Compare(tempZeroForCompare) == 0 {
		n = NewInt(0) // Remainder is 0
		bi.Zero()     // Quotient is 0 (bi is already 0, Zero() ensures canonical form)
		return
	}

	switch bi.Compare(x) {
	case 0:
		// if bi and x are equal, quotient is one, remainder is zero
		n = NewInt(0)
		bi.Set(NewInt(1)) // Quotient is 1
		return
	case -1:
		// if x is greater than bi,
		// then quotient is zero and remainder is bi
		n = NewInt(0)
		n.Set(bi) // Remainder is the original bi
		bi.Zero()   // Quotient is 0
		return
	}

	// Long division logic starts here
	// q will accumulate the quotient.
	// r will be the running remainder (currentDividendPart).
	q := NewInt(0) 
	r := NewInt(0) 

	// Iterate from most significant limb of dividend (bi)
	for i := len(bi.nat) - 1; i >= 0; i-- {
		// 1. Bring down the next limb from the dividend into r
		// r = (r << 16) + bi.nat[i]
		r.shift16(1)      // Make space by shifting left (logical left shift)
		r.nat[0] = bi.nat[i] // Set the least significant limb
		r.normalize()        // Important after potential leading zeros from shift if r was [0]

		// 2. Estimate how many times x fits into r (currentDividendPart)
		// q_limb will be the quotient digit for this step
		var q_limb uint16 = 0
		if r.Compare(x) >= 0 {
			// Binary search for q_limb: find largest ql such that NewInt(int(ql)).Mul(x) <= r.
			// ql is between 1 and 0xFFFF.
			var low uint32 = 1
			var high uint32 = 0xFFFF
			var current_ql_val uint16 = 0 // Best q_limb found so far in binary search

			for low <= high {
				mid := (low + high) / 2
				if mid == 0 { // Avoid infinite loop or issues if mid becomes 0
					break
				}
				ql_candidate_int := NewInt(int(mid))
				temp_product := NewInt(0)
				temp_product.Set(x)
				temp_product.Mul(ql_candidate_int)

				if temp_product.Compare(r) <= 0 {
					// This ql_candidate_int is a possible candidate (or too small)
					current_ql_val = uint16(mid)
					low = mid + 1 // Try for a larger q_limb
				} else {
					// This ql_candidate_int is too large
					high = mid - 1
				}
			}
			q_limb = current_ql_val // The largest one found
		}
		// q_limb is now the estimated (and refined) quotient limb for this step.
		// It could be 0 if r < x.

		// 3. Add q_limb to the overall quotient q
		// q = (q << 16) + q_limb
		q.shift16(1)
		q.nat[0] = q_limb
		// q.normalize() // Normalize q: [0,0,X] should become [X] if MSB, or [0,X]
		// Normalizing q at each step might be slightly inefficient but safe.
		// Consider normalizing q once at the end, but intermediate leading zeros
		// could affect Compare or other operations if they expect normalized numbers.
		// For now, let's normalize. If q starts as [0], shift makes it [0,0], then nat[0]=ql makes it [ql,0].
		// This needs to be [0,ql] for big-endian interpretation of how q is built.
		// The current shift16 prepends, so q.nat[0] is LSB. This is correct.
		// Example: q=0. q_limb=A -> q.shift16(1) -> q=[0,0]. q.nat[0]=A -> q=[A,0]. This is q=A.
		// Next, q_limb=B. q.shift16(1) -> q=[0,A,0]. q.nat[0]=B -> q=[B,A,0]. This is q = A*2^16 + B. Correct.
		q.normalize()


		// 4. Subtract (q_limb * x) from r
		if q_limb > 0 {
			temp_product_sub := NewInt(0)
			temp_product_sub.Set(x)
			temp_product_sub.Mul(NewInt(int(q_limb)))
			r.Sub(temp_product_sub)
			r.normalize() // Normalize remainder after subtraction
		}
	}

	// Finalization
	bi.Set(q) // bi becomes the quotient
	n = NewInt(0)
	n.Set(r)  // n is the final remainder
	n.normalize() // Ensure remainder is normalized

	return
}

// shift bi by x count of 16 bits words
func (bi *Int) shift16(count int) {
	for shift := 0; shift < count; shift++ {
		bi.nat = append([]uint16{0}, bi.nat...)
	}
}

// Zero resets a big integer to zero
func (bi *Int) Zero() {
	bi.nat = []uint16{0}
}

// normalize removes leading zero limbs from bi.nat.
// If bi.nat is empty or all zeros, it becomes [0].
// For example, [0,0,1,2,0,0] becomes [0,0,1,2] (assuming little-endian)
// No, for little-endian [1,2,0,0] becomes [1,2]. [0,0,0] becomes [0].
// [0,0,1,2] means 2*16^3 + 1*16^2.
// normalize should remove trailing zeros in the slice, which are leading zeros of the number.
func (bi *Int) normalize() {
	if len(bi.nat) == 0 {
		bi.nat = []uint16{0}
		return
	}
	i := len(bi.nat) - 1
	for i > 0 && bi.nat[i] == 0 { // Iterate from the end of the slice
		i--
	}
	bi.nat = bi.nat[:i+1] // Slice to the last non-zero limb
}

// Increment adds one to big integer
func (bi *Int) Increment() {
	bi.Add(NewInt(1))
}

// Decrement substracts one from big integer
func (bi *Int) Decrement() {
	bi.Sub(NewInt(1))
}

// Compare returns 1 if bi is greater than x, 0 if they
// are equal, and -1 if bi is smaller than x.
func (bi *Int) Compare(x *Int) (r int) {
	m := len(bi.nat)
	n := len(x.nat)
	if m != n || m == 0 {
		// compare the the length of the nat slices
		// to get a quick answer on which number is larger
		if m < n {
			return -1
		} else {
			return 1
		}
	}

	// if the nat len are equal, iterate over the nat limb
	// on bi until we find one that isn't identical to the
	// nat link of the same indice on x. Then compare those
	// two limbs to find out which is greater.
	var i int
	for i = m - 1; i > 0 && bi.nat[i] == x.nat[i]; i-- {
	}
	switch {
	case bi.nat[i] < x.nat[i]:
		return -1
	case bi.nat[i] > x.nat[i]:
		return 1
	}
	return 0 // bi and x are equal
}

// ModularExponentiation raises a big integer bi to the exponent x
// and reduces it modulo n, such as bi = bi^x mod n
func (bi *Int) ModularExponentiation(x *Int, modulus *Int) {
	/* from https://en.wikipedia.org/wiki/Modular_exponentiation#Memory-efficient_method
		if modulus = 1 then
	        return 0
	    c := 1
	    for e_prime = 0 to exponent-1 do
	        c := (c * base) mod modulus
	    return c
	*/
	if modulus.Compare(NewInt(1)) == 0 {
		bi.Zero()
		return
	}

	c := NewInt(1)
	//fmt.Printf("computing %x ^ %x mod %x\n", bi.Bytes(), x.Bytes(), modulus.Bytes())
	for e := NewInt(0); e.Compare(x) < 0; e.Increment() {
		//fmt.Printf("%x: %x * %x mod %x\n", e.Bytes(), c.Bytes(), base.Bytes(), modulus.Bytes())
		c.Mul(bi)
		// c.Div(modulus) modifies c to be the quotient and returns the remainder.
		// For modular exponentiation, we need c to become the remainder.
		remainder := c.Div(modulus)
		c.Set(remainder)
	}
	bi.Set(c)
}

// IsFermatPrime returns true if a given big integer is considered
// prime using Fermat's primality test
func (bi *Int) IsFermatPrime() bool {
	p := new(Int)
	p.Set(bi)
	pmin := new(Int)
	pmin.Set(bi)
	pmin.Decrement()
	for _, step := range []int{2, 3, 5, 7} {
		a := NewInt(step)
		a.ModularExponentiation(pmin, p)
		if a.Compare(NewInt(1)) != 0 {
			return false
		}
	}
	return true
}

// IsRabinMillerPrime implements primality test of bi using the
// Rabin-Miller algorithm
//func (bi *Int) IsRabinMillerPrime() bool {
//}
