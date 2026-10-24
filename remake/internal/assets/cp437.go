package assets

import "fmt"

// cp437High are the characters of the bytes 0x80..0xff of code page 437,
// the encoding of the names in MANA.DAT. The extract stores the names as
// UTF-8 with this table, so every byte comes back unchanged (STANDARD
// L\xdcTTICH of the original, a Latin-1 Ü, reads "L▄TTICH" there).
var cp437High = [128]rune{
	'Ç', 'ü', 'é', 'â', 'ä', 'à', 'å', 'ç', 'ê', 'ë', 'è', 'ï', 'î', 'ì', 'Ä', 'Å',
	'É', 'æ', 'Æ', 'ô', 'ö', 'ò', 'û', 'ù', 'ÿ', 'Ö', 'Ü', '¢', '£', '¥', '₧', 'ƒ',
	'á', 'í', 'ó', 'ú', 'ñ', 'Ñ', 'ª', 'º', '¿', '⌐', '¬', '½', '¼', '¡', '«', '»',
	'░', '▒', '▓', '│', '┤', '╡', '╢', '╖', '╕', '╣', '║', '╗', '╝', '╜', '╛', '┐',
	'└', '┴', '┬', '├', '─', '┼', '╞', '╟', '╚', '╔', '╩', '╦', '╠', '═', '╬', '╧',
	'╨', '╤', '╥', '╙', '╘', '╒', '╓', '╫', '╪', '┘', '┌', '█', '▄', '▌', '▐', '▀',
	'α', 'ß', 'Γ', 'π', 'Σ', 'σ', 'µ', 'τ', 'Φ', 'Θ', 'Ω', 'δ', '∞', 'φ', 'ε', '∩',
	'≡', '±', '≥', '≤', '⌠', '⌡', '÷', '≈', '°', '∙', '·', '√', 'ⁿ', '²', '■', ' ',
}

var cp437Bytes = func() map[rune]byte {
	m := map[rune]byte{}
	for i, r := range cp437High {
		m[r] = byte(0x80 + i)
	}
	return m
}()

// FromCP437 converts a CP437 string, a name of a club database, to UTF-8.
func FromCP437(s string) string {
	r := make([]rune, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] < 0x80 {
			r = append(r, rune(s[i]))
		} else {
			r = append(r, cp437High[s[i]-0x80])
		}
	}
	return string(r)
}

// ToCP437 converts a UTF-8 string to code page 437, the encoding of the
// names of a club database (Mana); characters CP437 does not have are an
// error.
func ToCP437(s string) (string, error) {
	b := make([]byte, 0, len(s))
	for _, r := range s {
		c, ok := cp437Bytes[r]
		switch {
		case r < 0x80:
			c = byte(r)
		case !ok:
			return "", fmt.Errorf("%q: %q is not in code page 437", s, r)
		}
		b = append(b, c)
	}
	return string(b), nil
}
