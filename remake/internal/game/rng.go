// Package game implements the rules of Bundesliga Manager Professional.
//
// The package is a port of the logic of BMMAIN.EXE. It keeps the data model
// of the original (club slots by id, leagues as id ranges, identity swaps for
// promotion and relegation) because many rules depend on it. Comments name
// the original function a piece of code was taken from (f_SSSS_OOOO), and
// docs/re/ holds the specifications they were reverse-engineered into.
package game

// Rand is the random number generator of the Microsoft C 6 runtime the
// original was built with, so that sequences of random decisions behave the
// same way.
type Rand struct {
	Seed uint32
}

// Next returns rand(): 0..32767.
func (r *Rand) Next() int {
	r.Seed = r.Seed*214013 + 2531011
	return int(r.Seed>>16) & 0x7fff
}

// Between returns a number in [a, b] like f_076b_0cc7: rand() % (b-a+1) + a.
// Like the original it uses a truncating C modulo, so b < a gives odd but
// defined results instead of a panic.
func (r *Rand) Between(a, b int) int {
	n := b - a + 1
	if n == 0 {
		return a
	}
	return r.Next()%n + a
}

// Between16 is Between with the 16-bit int arguments and arithmetic of the
// original, for the places where the original passes values that do not
// fit into 16 bits.
func (r *Rand) Between16(a, b int) int {
	a16, b16 := int16(a), int16(b)
	n := int16(b16 - a16 + 1)
	if n == 0 {
		return int(a16)
	}
	return int(int16(int16(r.Next())%n + a16))
}
