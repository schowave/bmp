// Package assets reads the files of the original game: pictures, palettes,
// the bitmap fonts embedded in BMMAIN.EXE and the club database MANA.DAT.
//
// The files are read from the directory of an installed copy of the game
// or, without one, from the copy built into the binary (package embedded).
package assets

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Image is the load image of an MZ executable: the bytes as DOS would place
// them in memory, starting at segment 0.
type Image struct {
	Data []byte
	CS   uint16
	IP   uint16
}

// Unexepack reads an MZ executable compressed with Microsoft EXEPACK and
// returns its decompressed load image. Relocations are not applied; segment
// values in the image are relative to load segment 0.
func Unexepack(exe []byte) (*Image, error) {
	if len(exe) < 0x1c || string(exe[:2]) != "MZ" {
		return nil, errors.New("not an MZ executable")
	}
	u16 := func(b []byte, off int) uint16 { return binary.LittleEndian.Uint16(b[off:]) }
	cblp, cp := u16(exe, 2), u16(exe, 4)
	hdr := int(u16(exe, 8)) * 16
	ip, cs := u16(exe, 0x14), u16(exe, 0x16)
	size := int(cp) * 512
	if cblp != 0 {
		size -= 512 - int(cblp)
	}
	if size > len(exe) || hdr > size {
		return nil, errors.New("truncated MZ executable")
	}
	image := exe[hdr:size]
	epos := int(cs) * 16
	if epos+18 > len(image) {
		return nil, errors.New("no EXEPACK header")
	}
	// The header is 16 bytes, or 18 when it carries skip_len (then IP is 0x12).
	realIP, realCS := u16(image, epos), u16(image, epos+2)
	destLen := int(u16(image, epos+12))
	skipLen := 1
	sigOff := epos + 14
	if ip == 0x12 {
		skipLen = int(u16(image, epos+14))
		sigOff = epos + 16
	}
	if string(image[sigOff:sigOff+2]) != "RB" {
		return nil, errors.New("not packed with EXEPACK")
	}
	end := epos - (skipLen-1)*16
	if end < 0 || end > len(image) {
		return nil, errors.New("bad EXEPACK skip length")
	}
	src := image[:end]
	dst := make([]byte, destLen*16)
	s := len(src)
	for s > 0 && src[s-1] == 0xff {
		s--
	}
	d := len(dst)
	for {
		if s < 3 {
			return nil, errors.New("EXEPACK stream underflow")
		}
		cmd := src[s-1]
		length := int(src[s-3]) | int(src[s-2])<<8
		s -= 3
		switch cmd & 0xfe {
		case 0xb0:
			if s < 1 || d < length {
				return nil, errors.New("EXEPACK fill out of range")
			}
			v := src[s-1]
			s--
			for i := 0; i < length; i++ {
				dst[d-1-i] = v
			}
			d -= length
		case 0xb2:
			if s < length || d < length {
				return nil, errors.New("EXEPACK copy out of range")
			}
			copy(dst[d-length:d], src[s-length:s])
			d -= length
			s -= length
		default:
			return nil, fmt.Errorf("bad EXEPACK command %#x", cmd)
		}
		if cmd&1 != 0 {
			break
		}
	}
	// Everything below the last decompressed byte was stored uncompressed.
	copy(dst[:s], src[:s])
	return &Image{Data: dst, CS: realCS, IP: realIP}, nil
}

// cString returns the NUL-terminated string at off.
func cString(b []byte, off int) string {
	if off < 0 || off >= len(b) {
		return ""
	}
	end := bytes.IndexByte(b[off:], 0)
	if end < 0 {
		return string(b[off:])
	}
	return string(b[off : off+end])
}
