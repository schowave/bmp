package ui

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ParseScript reads a development script for -script, one command per
// line, in canvas coordinates:
//
//	click X Y     left click
//	rclick X Y    right click
//	move X Y      move the pointer
//	click NAME    left click on a named target (App.target), also rclick
//	              and move
//	dump FILE     write the running game as a save file
//	type TEXT     type characters
//	key NAME      enter, escape or backspace
//	wait N        N frames without input
//	shot FILE     save the canvas as PNG
//
// Every command takes one frame, followed by a frame without buttons.
func ParseScript(r io.Reader) ([]ScriptStep, error) {
	var steps []ScriptStep
	x, y := 160, 120
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		f := strings.Fields(sc.Text())
		if len(f) == 0 || strings.HasPrefix(f[0], "#") {
			continue
		}
		num := func(i int) (int, error) {
			if len(f) <= i {
				return 0, fmt.Errorf("line %d: missing argument", n)
			}
			return strconv.Atoi(f[i])
		}
		in := Input{X: x, Y: y}
		switch f[0] {
		case "click", "rclick", "move":
			if len(f) == 2 {
				in = Input{Left: f[0] == "click", Right: f[0] == "rclick", LeftHeld: f[0] == "click"}
				steps = append(steps, ScriptStep{In: in, Target: f[1]}, ScriptStep{Target: f[1]})
				continue
			}
			var err error
			if x, err = num(1); err != nil {
				return nil, err
			}
			if y, err = num(2); err != nil {
				return nil, err
			}
			in = Input{X: x, Y: y, Left: f[0] == "click", Right: f[0] == "rclick", LeftHeld: f[0] == "click"}
		case "type":
			in.Chars = []rune(strings.Join(f[1:], " "))
		case "key":
			switch strings.ToLower(strings.Join(f[1:], "")) {
			case "enter":
				in.Enter = true
			case "escape":
				in.Escape = true
			case "backspace":
				in.Backspace = true
			default:
				return nil, fmt.Errorf("line %d: unknown key", n)
			}
		case "wait":
			k, err := num(1)
			if err != nil {
				return nil, err
			}
			for ; k > 0; k-- {
				steps = append(steps, ScriptStep{In: Input{X: x, Y: y}})
			}
			continue
		case "dump":
			if len(f) < 2 {
				return nil, fmt.Errorf("line %d: missing file", n)
			}
			steps = append(steps, ScriptStep{In: Input{X: x, Y: y}, Dump: f[1]})
			continue
		case "shot":
			if len(f) < 2 {
				return nil, fmt.Errorf("line %d: missing file", n)
			}
			steps = append(steps, ScriptStep{In: Input{X: x, Y: y}, Shot: f[1]})
			continue
		default:
			return nil, fmt.Errorf("line %d: unknown command %q", n, f[0])
		}
		steps = append(steps, ScriptStep{In: in}, ScriptStep{In: Input{X: x, Y: y}})
	}
	return steps, sc.Err()
}
