package game

import (
	"strconv"
	"strings"
)

// Message is a note for a manager: the original shows them as message
// boxes during the turn or keeps them in a mailbox.
type Message struct {
	Text string
}

// notify adds a message for manager mi.
func (s *State) notify(mi int, text string) {
	if mi < 0 || mi >= len(s.Managers) {
		return
	}
	m := s.Managers[mi]
	m.Inbox = append(m.Inbox, Message{Text: text})
}

// notifyTeam adds a message for the manager of club id, if any.
func (s *State) notifyTeam(id int, text string) {
	if m := s.ManagerOf(id); m != nil {
		s.notify(s.managerIndex(m), text)
	}
}

// TakeInbox returns and clears the messages of manager mi.
func (s *State) TakeInbox(mi int) []Message {
	m := s.Managers[mi]
	msgs := m.Inbox
	m.Inbox = nil
	return msgs
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
