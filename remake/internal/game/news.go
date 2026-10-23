package game

import (
	"strconv"
	"strings"
)

// Message is a note for a manager in the message panel of the main menu
// (f_3091_0190): the day it was written and the text. Person is the player
// of a message a squad or market slot points to (slot +0x30: an offer of a
// club, a player asking for a longer contract), 0 for the others; it goes
// when the slot lets it go (endMessage).
type Message struct {
	Day    int
	Text   string
	Person int `json:",omitempty"`
}

// MaxMessages is the length of a manager's message list (F:0x5524, 20 per
// manager).
const MaxMessages = 20

// notify adds a message for manager mi at the front of the list; the panel
// opens by itself at the next turn (F:0x56e2).
func (s *State) notify(mi int, text string) {
	if mi < 0 || mi >= len(s.Managers) {
		return
	}
	m := s.Managers[mi]
	m.Inbox = append([]Message{{Day: s.Day, Text: text}}, m.Inbox...)
	if len(m.Inbox) > MaxMessages {
		m.Inbox = m.Inbox[:MaxMessages]
	}
	m.NewMessages = true
}

// notifyFor adds a message about the player person that his slot points to
// (Slot.Message).
func (s *State) notifyFor(mi, person int, text string) {
	s.notify(mi, text)
	if mi >= 0 && mi < len(s.Managers) && len(s.Managers[mi].Inbox) > 0 {
		s.Managers[mi].Inbox[0].Person = person
	}
}

// endMessage lets the message of a slot go: the offer expired or was
// answered, the request was withdrawn. As the original deletes the message
// the slot points to (+0x30), it leaves the message panel too.
func (s *State) endMessage(sl *Slot) {
	sl.Message = false
	for _, m := range s.Managers {
		kept := m.Inbox[:0]
		for _, msg := range m.Inbox {
			if msg.Person == 0 || msg.Person != sl.Person {
				kept = append(kept, msg)
			}
		}
		m.Inbox = kept
	}
}

// TakeInbox returns and clears the messages of manager mi.
func (s *State) TakeInbox(mi int) []Message {
	m := s.Managers[mi]
	msgs := m.Inbox
	m.Inbox = nil
	m.NewMessages = false
	return msgs
}

// Messages returns the message list of manager mi, the newest first.
func (s *State) Messages(mi int) []Message { return s.Managers[mi].Inbox }

// TakeNewMessages reports whether manager mi got messages since the panel
// was last shown, and clears the flag.
func (s *State) TakeNewMessages(mi int) bool {
	m := s.Managers[mi]
	n := m.NewMessages
	m.NewMessages = false
	return n
}

// FormatMoney formats an amount with dots as thousands separators like
// the original (f_076b_0681): 1500000 -> "1.500.000".
func FormatMoney(v int64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	digits := strconv.FormatInt(v, 10)
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i, c := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return b.String()
}
