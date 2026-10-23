package game

// The German letters of code page 437, the encoding of the original's files
// and fonts.
var (
	cp437Letters = map[byte]rune{0x8e: 'Ä', 0x99: 'Ö', 0x9a: 'Ü', 0x84: 'ä', 0x94: 'ö', 0x81: 'ü', 0xe1: 'ß'}
	cp437Bytes   = map[rune]byte{'Ä': 0x8e, 'Ö': 0x99, 'Ü': 0x9a, 'ä': 0x84, 'ö': 0x94, 'ü': 0x81, 'ß': 0xe1}
)

// FromCP437 converts a CP437 string of the original's files to UTF-8.
func FromCP437(s string) string {
	var r []rune
	for i := 0; i < len(s); i++ {
		if u, ok := cp437Letters[s[i]]; ok {
			r = append(r, u)
		} else {
			r = append(r, rune(s[i]))
		}
	}
	return string(r)
}

// ToCP437 converts a UTF-8 string to CP437: '?' for anything but ASCII and
// the German letters.
func ToCP437(s string) []byte {
	b := make([]byte, 0, len(s))
	for _, r := range s {
		c, ok := cp437Bytes[r]
		switch {
		case r < 0x80:
			c = byte(r)
		case !ok:
			c = '?'
		}
		b = append(b, c)
	}
	return b
}
