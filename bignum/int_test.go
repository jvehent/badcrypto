package bignum

import "testing"

func TestNewInt(t *testing.T) {
	var testnums = []int{3, -3, 0, 9223372036854775807, -9223372036854775807}
	for n, num := range testnums {
		bi := NewInt(num)
		if bi.neg && num > 0 {
			t.Fatalf("testcase %d expected a positive integer but got a negative", n)
		}
		if !bi.neg && num < 0 {
			t.Fatalf("testcase %d expected a negative integer but got a positive", n)
		}
		if bi.natlen == 0 {
			t.Fatalf("testcase %d has zero length", n)
		}
		if bi.ToInt() != num {
			t.Fatalf("testcase %d expected to retrieve integer %d, but got %v", n, num, bi.ToInt())
		}
	}
}
