package game

import (
	"runtime"
	"strings"
	"sync"
	"weak"
)

// NoticeKind is the way the original shows a note at once instead of
// adding it to the message panel.
type NoticeKind int

const (
	// NoticeAlert is the 4-line box without title and with an OKAY
	// button of f_3091_0e3a; Lines are the four lines.
	NoticeAlert NoticeKind = iota
	// NoticeCard is the card of a special date (f_1a58_2a06): Title,
	// Lines in the normal font, Small (the Christmas amounts) in the small
	// font; Birthday cards close by themselves after a minute.
	NoticeCard
	// NoticeTrophy is the title screen of f_17c2_274d: Lines are the club,
	// "gewinnt die"/"gewinnt den" and the title; Cup 0 is the championship
	// (picture PIC/50.VGA), 1..4 the cups (51..54).
	NoticeTrophy
	// NoticeSeasonEnd is the season report of f_0cb5_0012: Title ("Sie
	// beendeten Ihre Saison"), Lines[0] the place, Heading ("Ereignisse"),
	// Lines[1:] the events. Dialogs are shown over it while it is drawn,
	// Popups after the click that closes it.
	NoticeSeasonEnd
	// NoticePopup is the message panel of f_3091_05c4 called with a single
	// text: Lines are the message, Day its date.
	NoticePopup
)

// Backdrop is what the original shows behind an alert or a popup.
type Backdrop int

const (
	// BackdropNone: the alert is drawn over the current screen.
	BackdropNone Backdrop = iota
	// BackdropMenu: the main menu of the manager whose turn it is (the goal
	// messages of f_112a_314d are shown by the main menu f_08bc_1415).
	BackdropMenu
	// BackdropDesk: the marble with the manager's face, name and logo at
	// the top right (f_08bc_3b92 + f_3091_185c, f_14a4_260a).
	BackdropDesk
)

// Notice is a note the UI has to show at once, in the order of the queue.
type Notice struct {
	Kind     NoticeKind
	Manager  int // index of the manager it is for, -1 = everybody
	Backdrop Backdrop
	Title    string
	Heading  string
	Lines    []string
	Small    []string
	Day      int  // date of a card or popup
	Birthday bool // card: closes after a minute, "Das war keine Minute !"
	Cup      int  // trophy: 0 championship, 1..4 cup
	Dialogs  []NoticeDialog
	Popups   []Notice
}

// NoticeDialog is an alert shown over a season report when At lines of
// the report (Lines[:At]) are drawn.
type NoticeDialog struct {
	At    int
	Lines []string
}

// The queues live beside the State until it gets a field of its own; a
// weak key lets a State be collected with its queue.
var noticeQueues struct {
	sync.Mutex
	m map[weak.Pointer[State]][]Notice
}

func (s *State) noticeKey() weak.Pointer[State] {
	k := weak.Make(s)
	if noticeQueues.m == nil {
		noticeQueues.m = map[weak.Pointer[State]][]Notice{}
	}
	if _, ok := noticeQueues.m[k]; !ok {
		noticeQueues.m[k] = nil
		runtime.AddCleanup(s, func(k weak.Pointer[State]) {
			noticeQueues.Lock()
			delete(noticeQueues.m, k)
			noticeQueues.Unlock()
		}, k)
	}
	return k
}

// AddNotice queues a notice for the UI.
func (s *State) AddNotice(n Notice) {
	noticeQueues.Lock()
	defer noticeQueues.Unlock()
	k := s.noticeKey()
	noticeQueues.m[k] = append(noticeQueues.m[k], n)
}

// TakeNotices returns the queued notices, the oldest first, and clears
// the queue.
func (s *State) TakeNotices() []Notice {
	noticeQueues.Lock()
	defer noticeQueues.Unlock()
	k := weak.Make(s)
	q := noticeQueues.m[k]
	if q != nil {
		noticeQueues.m[k] = []Notice{}
	}
	return q
}

// HasNotices reports whether notices are waiting.
func (s *State) HasNotices() bool {
	noticeQueues.Lock()
	defer noticeQueues.Unlock()
	return len(noticeQueues.m[weak.Make(s)]) > 0
}

// lastNotice returns the newest queued notice of manager mi of kind k.
func (s *State) lastNotice(mi int, k NoticeKind) *Notice {
	noticeQueues.Lock()
	defer noticeQueues.Unlock()
	q := noticeQueues.m[weak.Make(s)]
	for i := len(q) - 1; i >= 0; i-- {
		if q[i].Kind == k && q[i].Manager == mi {
			return &q[i]
		}
	}
	return nil
}

// lines splits a message text of msgtexts.go into its lines.
func lines(text string) []string { return strings.Split(text, "\n") }

// alert queues a 4-line OKAY box (f_3091_0e3a) for manager mi.
func (s *State) alert(mi int, b Backdrop, text string) {
	s.AddNotice(Notice{Kind: NoticeAlert, Manager: mi, Backdrop: b, Lines: lines(text)})
}

// popup queues a message of the panel that the original shows at once in
// the panel and removes again (f_3091_0190(1, ...) / f_3091_05c4).
func (s *State) popup(mi int, b Backdrop, text string) {
	s.AddNotice(Notice{Kind: NoticePopup, Manager: mi, Backdrop: b, Lines: lines(text), Day: s.Day})
}

// seasonScreen starts the season report of manager mi.
func (s *State) seasonScreen(mi, pos, league int) {
	s.AddNotice(Notice{Kind: NoticeSeasonEnd, Manager: mi, Backdrop: BackdropDesk,
		Title: msgSeasonHead, Heading: msgSeasonEvents, Lines: []string{msgSeasonPlace(pos, league, s.NameYear(s.StartYear))}})
}

// seasonLine adds an event line to the season report of manager mi.
func (s *State) seasonLine(mi int, text string) {
	if n := s.lastNotice(mi, NoticeSeasonEnd); n != nil {
		n.Lines = append(n.Lines, text)
	}
}

// seasonDialog shows an alert over the season report of manager mi as it
// is drawn so far.
func (s *State) seasonDialog(mi int, text string) {
	if n := s.lastNotice(mi, NoticeSeasonEnd); n != nil {
		n.Dialogs = append(n.Dialogs, NoticeDialog{At: len(n.Lines), Lines: lines(text)})
	}
}

// seasonDone ends the event lines of the report: without events it says
// "Keine besonderen Ereignisse...".
func (s *State) seasonDone(mi int) {
	if n := s.lastNotice(mi, NoticeSeasonEnd); n != nil && len(n.Lines) == 1 {
		n.Lines = append(n.Lines, msgNoEvents)
	}
}

// seasonPopup shows a panel message over the season report of manager mi
// after its click (the retirements of f_0cb5_0012), or on its own.
func (s *State) seasonPopup(mi int, text string) {
	if n := s.lastNotice(mi, NoticeSeasonEnd); n != nil {
		n.Popups = append(n.Popups, Notice{Kind: NoticePopup, Manager: mi, Lines: lines(text), Day: s.Day})
		return
	}
	s.popup(mi, BackdropDesk, text)
}

// trophy queues the title screen of f_17c2_274d for manager mi.
func (s *State) trophy(mi, trophy int) {
	s.AddNotice(Notice{Kind: NoticeTrophy, Manager: mi, Cup: trophy,
		Lines: lines(msgTrophy(s.Teams[s.Managers[mi].Team].Name, trophy))})
}

// popupWindow reports the days on which f_0000_20e1 shows the completion
// of a construction at once: the winter break (day 129..206) and after the
// last matchday (from day 323).
func (s *State) popupWindow() bool {
	return (s.Day > 128 && s.Day < 207) || s.Day > LastSeasonDay
}

// SampleNotices returns one notice of every kind with the texts of the
// original, for demos and screenshots.
func SampleNotices(s *State) []Notice {
	m := s.Managers[0]
	club := s.Teams[m.Team].Name
	card := Notice{Kind: NoticeCard, Manager: -1, Title: msgChristmasTitle, Day: 148} // 24 December
	full := card
	full.Lines, full.Small = msgChristmasLines[:], msgChristmasAmounts(450000, s.NameYear(s.StartYear))
	bday := Notice{Kind: NoticeCard, Manager: -1, Title: msgBirthdayTitle, Lines: msgBirthday(1),
		Small: msgBirthdayShame, Day: 106, Birthday: true} // 12 November
	season := Notice{Kind: NoticeSeasonEnd, Manager: 0, Backdrop: BackdropDesk, Title: msgSeasonHead,
		Heading: msgSeasonEvents, Lines: []string{msgSeasonPlace(1, 1, s.NameYear(s.StartYear)), msgPromoted2},
		Dialogs: []NoticeDialog{{At: 2, Lines: lines(msgAdsFree)}}}
	season.Lines = append(season.Lines, msgTopScorer, msgEurope, msgYouthPlayer)
	season.Dialogs = append(season.Dialogs, NoticeDialog{At: 5, Lines: lines(msgJoke1)}, NoticeDialog{At: 5, Lines: lines(msgJoke2)})
	season.Popups = []Notice{{Kind: NoticePopup, Manager: 0, Lines: lines(msgRetires("TRAUTNER", 34)), Day: s.Day}}
	return []Notice{
		{Kind: NoticeAlert, Manager: 0, Backdrop: BackdropDesk, Lines: lines(msgCreditDue("der Bank"))},
		{Kind: NoticeAlert, Manager: 0, Backdrop: BackdropMenu, Lines: lines(msgWinterBreak)},
		{Kind: NoticeAlert, Manager: 0, Backdrop: BackdropMenu, Lines: lines(msgRelegation)},
		{Kind: NoticeAlert, Manager: 0, Backdrop: BackdropMenu, Lines: lines(msgGoalLost(m.Name, club, "MEISTER zu werden."))},
		{Kind: NoticeAlert, Manager: 0, Backdrop: BackdropMenu, Lines: lines(msgGoalWon(m.Name, club, "den KLASSENERHALT"))},
		{Kind: NoticeAlert, Manager: 0, Lines: lines(msgForfeit(m.Name))},
		full, card, bday,
		{Kind: NoticeTrophy, Manager: 0, Lines: lines(msgTrophy(club, 0))},
		{Kind: NoticeTrophy, Manager: 0, Cup: int(CupDFB) + 1, Lines: lines(msgTrophy(club, int(CupDFB)+1))},
		season,
		{Kind: NoticePopup, Manager: 0, Backdrop: BackdropDesk, Lines: lines(msgCompletion(3)), Day: s.Day},
	}
}
