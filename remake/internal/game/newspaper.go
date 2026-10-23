package game

import (
	"strconv"
	"strings"
)

// The newspaper "Sportnachrichten" shown after a league match day for every
// human manager whose team played (f_2e3a_23aa -> f_2e3a_0ea3). It is built
// from the match report of the manager (the record F:0x90ca + m*0x9a that
// f_2e3a_223e fills during the match; the remake rebuilds it from the
// events of the match day), the grades of the players and the ratings the
// match engine collected in Slot.Rating.

// Layout constants of the newspaper (f_2e3a_0c0b, f_2e3a_000e).
const (
	NewsLeft         = 0x0d  // left edge of all text
	NewsArticleTop   = 0x3c  // baseline of the first article line
	NewsArticleWidth = 0xb4  // width of the article column
	NewsBoxWidth     = 0x127 // width of the line-up, goals and cards lines
	NewsLineStep     = 7     // distance of the lines
	newsArticleStop  = 0x94  // no paragraph starts below this line
	newsLineupTop    = 0xbb
	newsGoalsTop     = 0xd1
	newsYellowTop    = 0xe6
	newsRedTop       = 0xed
	NewsHeadlineY1   = 0x29 // baselines of the two headline lines
	NewsHeadlineY2   = 0x33
	NewsFirstPhoto   = 200 // PIC/200..229.VGA
	NewsPhotos       = 30
)

// NewsLine is one line of text of the newspaper. Lines with Split are drawn
// like f_2e3a_060a: the text up to the first ':' in colour 6, the rest in
// colour 14 (all of it when there is no ':'); the others in colour 0.
type NewsLine struct {
	X, Y  int // Y is the baseline
	Text  string
	Split bool
}

// Newspaper is the content of the newspaper page of one manager.
type Newspaper struct {
	Photo    int // picture number of the photo
	Headline [2]string
	Article  []NewsLine // small font, justified column
	Box      []NewsLine // line-up with grades, goals, cards
	Grades   []NewsGrade
}

// NewsGrade is the grade (1 sehr gut .. 6 ungenügend) of a player of the
// line-up after the match.
type NewsGrade struct {
	Slot  int
	Name  string
	Grade int
}

// newsChance is a goal or a missed chance of the match report.
type newsChance struct {
	minute int
	opp    bool   // the opponent's chance ('d' in the original)
	name   string // own scorer or shooter
}

// newsReport is the match report record of a manager.
type newsReport struct {
	own, opp           int // team ids (+0x0e, +0x0f)
	gf, ga             int // goals (+4, +5)
	chances            int // chances of both teams (+6)
	diff               int // goal difference (+9)
	minDiff, maxDiff   int // +0x0a, +0x0b
	minScore, maxScore [2]int
	state              int // +0x11: 0 draw, 1 ahead, 2 behind
	soldOut            bool
	attendance         int64 // 0 for away matches
	hero, loser        int   // slot+1 of the best and the worst player
	goals, misses      []newsChance
	yellow             []string
	red                string
	sel                [6]int // selectors F:0x579b+D of the templates
}

// Paragraph categories (counts at D:0x9350) and headline groups (counts at
// D:0x9322).
var (
	newsParagraphCounts = [...]int{8, 5, 4, 8, 5, 7, 4, 7, 8, 10, 4, 4, 7, 6, 6, 5, 1, 1, 2, 7, 3, 2, 2, 3, 4}
	newsHeadlineGroups  = [...]int{4, 2, 3, 4, 2, 3, 1, 2, 4, 3, 2, 2, 7, 4, 5, 3, 3, 3, 3, 3, 2, 2, 2}
	newsWeights         = [...]int{3, 4, 5, 7, 9, 10, 9, 7, 5, 4, 3} // D:0x937e
)

// HasNewspaper reports whether manager mi gets a newspaper after this match
// day: the option is on, it was a league match day (or replays) and his team
// played.
func (md *MatchDay) HasNewspaper(mi int) bool {
	if !md.s.Display.Newspaper || md.cupMask != 0 || mi < 0 || mi >= len(md.humans) {
		return false
	}
	hm := &md.humans[mi]
	return hm.plays && md.Fixtures[hm.fixture].Forfeit == 0
}

// MatchGrade is the grade of slot i of manager mi after a match (the
// formula of f_2e3a_23aa).
func (s *State) MatchGrade(mi, i int) int {
	sl := &s.Squads[mi][i]
	r := clamp(int(int8(sl.Rating)), -12, 36)
	g := ((sl.Skill[0]+sl.Skill[1])/(-33)-abs(30-sl.Skill[2])/6+12)*2 +
		((-(r+12))/6+6)*4 + sl.Fatigue/33 + s.posDiff(sl)/5 + s.sideDiff(sl)
	return clamp(g/11, 1, 6)
}

// Newspaper builds the newspaper of manager mi after the match day
// (f_2e3a_0ea3). It draws random numbers like the original, so it must be
// called once per manager, in manager order, right after the match day.
// width measures a text in the small font of the original (micro4), which
// the layout of the justified lines depends on. It returns nil when the
// manager gets no newspaper.
func (md *MatchDay) Newspaper(mi int, width func(string) int) *Newspaper {
	if !md.HasNewspaper(mi) {
		return nil
	}
	s := md.s
	r := &s.Rand
	rep := md.newsReport(mi)
	n := s.countLineup(mi, 0)
	grades := make([]int, n)
	for i := 0; i < n; i++ {
		if l := s.Squads[mi][i].Lineup; l >= 1 && l <= 11 {
			grades[i] = s.MatchGrade(mi, i)
		}
	}
	np := &Newspaper{Photo: NewsFirstPhoto + r.Between(0, NewsPhotos-1)}

	// Course of the match: goals and chances weighted around their minute.
	var mom [120]int
	earlyOwn, earlyOpp, late := false, false, false
	own, opp := 0, 0
	addMoment := func(c newsChance, count *int) {
		for j, k := 0, c.minute-5; k < c.minute+6; j, k = j+1, k+1 {
			if k <= 0 || k >= 120 {
				continue
			}
			if c.opp {
				mom[k] -= newsWeights[j]
			} else {
				mom[k] += newsWeights[j]
			}
			if count != nil && k == c.minute {
				*count++
			}
		}
	}
	for i, g := range rep.goals {
		last := len(rep.goals)-i == 1
		if g.opp {
			if g.minute < 8 && !earlyOwn {
				earlyOpp = true
			}
			if last && g.minute > 80 && (opp == own || own-opp == 1) {
				late = true
			}
			opp++
		} else {
			if g.minute < 8 && !earlyOpp {
				earlyOwn = true
			}
			if last && g.minute > 80 && (opp == own || opp-own == 1) {
				late = true
			}
			own++
		}
		addMoment(g, nil)
	}
	// Best and worst player.
	for i := 0; i < n; i++ {
		sl := &s.Squads[mi][i]
		rt := int(int8(sl.Rating))
		if s.Persons[sl.Person].Pos != 0 && rt > 25 {
			rep.hero = i + 1
			grades[i] = max(grades[i]-r.Between(1, 2), 1)
		}
		if rt < -15 && rep.loser == 0 {
			rep.loser = i + 1
			grades[i] = min(grades[i]+r.Between(1, 2), 6)
		}
	}
	ownMiss, oppMiss := 0, 0
	for _, c := range rep.misses {
		if c.opp {
			addMoment(c, &oppMiss)
		} else {
			addMoment(c, &ownMiss)
		}
	}
	sum := 0
	for _, v := range mom {
		sum += v
	}
	sum /= 120
	if sum > 2 {
		sum = 2
	}

	home := rep.attendance != 0
	rep.sel[1] = 2 - boolInt(home)
	rep.chances = clamp(rep.chances, 3, 10)
	rep.sel[2] = min(boolInt(rep.state == 1)+(rep.chances-3)/2+1, 4)
	rep.sel[3] = rep.state + 1
	str := func(t int) int {
		return (s.strengthAt(t, 1, 0) + s.strengthAt(t, 0, 0) + s.strengthAt(t, 2, 0)) * 2 / 15
	}
	ownS, oppS := str(rep.own), str(rep.opp)
	rep.sel[4] = 1
	if rep.state == 1 || rep.state == 0 && oppS > ownS || rep.state == 0 && !home && ownS+10 < oppS {
		rep.sel[4] = 2
	}
	rep.sel[5] = 2
	switch {
	case sum > 0:
		rep.sel[5] = 1
	case sum < 0:
		r.Between(0, 1)
	default:
		if r.Between(0, 1) != 0 {
			rep.sel[5] = 1
		}
	}

	// Paragraphs of the article.
	var first [len(newsParagraphCounts)]int
	for k := 1; k < len(first); k++ {
		first[k] = first[k-1] + newsParagraphCounts[k-1]
	}
	var paras []int
	para := func(k int) { paras = append(paras, r.Between(first[k], first[k]+newsParagraphCounts[k]-1)) }
	state := rep.state
	para(0)
	if home {
		para(1)
	} else {
		para(2)
	}
	if r.Between(0, 1) != 0 && len(rep.yellow) > 2 {
		para(3)
	}
	if r.Between(0, 1) != 0 && len(rep.yellow) == 0 {
		para(4)
	}
	if rep.hero != 0 && state == 1 {
		para(10)
	}
	if rep.loser != 0 && state == 2 {
		para(11)
	}
	para(5)
	if late {
		para(18)
	}
	if ownMiss > 3 && rep.diff < 2 {
		para(12)
	}
	if earlyOpp {
		para(16)
	}
	if earlyOwn {
		para(17)
	}
	if ownMiss < 2 && state == 1 {
		para(13)
	}
	if state == 2 {
		para(19)
	}
	if state == 1 {
		para(7)
	}

	// Headline: every group whose condition holds may replace the one
	// chosen so far (f_2e3a_21ef; the original stores 1 instead of the
	// drawn priority, so any later group wins unless it draws a 1).
	headline, best, base, group := 0, 0, 0, 0
	head := func(ok bool, prio int) {
		if ok {
			if r.Between(1, prio) > best {
				best = 1
				headline = r.Between(base, base+newsHeadlineGroups[group]-1)
			}
		}
		base += newsHeadlineGroups[group]
		group++
	}
	d := rep.diff
	absSum := abs(sum)
	head(d > 3, 5)
	head(d > 0 && d < 2, 5)
	head(d > 0 && d < 4, 5)
	head(d < -3, 5)
	head(d < 0 && d > -2, 5)
	head(d < 0 && d > -4, 5)
	head(d == 0, 2)
	head(rep.soldOut, 3)
	if rep.soldOut {
		para(6)
	}
	if rep.minDiff != 0 && state != 2 {
		para(14)
	}
	if rep.maxDiff != 0 && state != 1 && d > -2 {
		para(15)
	}
	head(rep.minDiff != 0 && state == 1, 5)
	head(rep.maxDiff != 0 && state == 2, 5)
	// The original tests the deficit for the texts about a lost lead and
	// the other way round.
	head(rep.minDiff < -1 && state == 0, 4)
	head(rep.maxDiff > 1 && state == 0, 4)
	head(rep.hero != 0 && state == 1, 4)
	head(rep.chances > 7 && absSum < 3, 3)
	head(rep.chances < 4 && absSum < 3, 3)
	head(rep.loser != 0 && state == 2, 4)
	head(rep.sel[5] == 1 && state == 2, 4)
	head(rep.sel[5] == 2 && state == 1, 4)
	head(rep.sel[5] == 1 && state == 1, 3)
	if rep.sel[5] == 1 && state == 1 {
		para(8)
	}
	head(rep.sel[5] == 2 && state == 2, 3)
	if rep.sel[5] == 2 && state == 2 {
		para(9)
	}
	head(absSum < 2 && state == 0, 4)
	head(rep.sel[5] == 2 && state == 0, 3)
	head(rep.sel[5] == 1 && state == 0, 3)

	ex := &newsExpander{md: md, mi: mi, rep: rep}
	t := newsHeadlines[headline]
	l1, p := ex.expand(t, 0)
	l2, _ := ex.expand(t, p)
	np.Headline = [2]string{l1, l2}

	// Article (f_2e3a_0c0b, f_2e3a_0d33).
	w := &newsWriter{width: width}
	w.start(NewsArticleWidth, NewsLeft, NewsArticleTop)
	for _, k := range paras {
		text, _ := ex.expand(newsParagraphs[k], 0)
		y := w.y
		for text != "" {
			i := strings.IndexByte(text, ' ')
			if i < 0 {
				y = w.word(text, false)
				break
			}
			y = w.word(text[:i], false)
			text = text[i+1:]
		}
		if y > newsArticleStop {
			break
		}
	}
	w.flush(false)
	np.Article = w.lines

	// Line-up, goals and cards (f_2e3a_000e).
	b := &newsWriter{width: width}
	b.start(NewsBoxWidth, NewsLeft, newsLineupTop)
	b.word(s.Teams[rep.own].Name+": ", true)
	for i := 0; i < n; i++ {
		sl := &s.Squads[mi][i]
		if sl.Lineup < 1 || sl.Lineup > 11 {
			continue
		}
		name := s.Persons[sl.Person].Name
		np.Grades = append(np.Grades, NewsGrade{Slot: i, Name: name, Grade: grades[i]})
		b.word(name+" ("+strconv.Itoa(grades[i])+"),", true)
	}
	b.flush(true)
	b.start(NewsBoxWidth, NewsLeft, newsGoalsTop)
	b.word("TORE: ", true)
	if len(rep.goals) == 0 {
		b.word("FEHLANZEIGE", true)
	} else {
		h, a := 0, 0
		for i, g := range rep.goals {
			if g.opp == home {
				a++
			} else {
				h++
			}
			t := strconv.Itoa(h) + ":" + strconv.Itoa(a) + " " + g.name + " (" + strconv.Itoa(g.minute) + ".MIN)"
			if i < len(rep.goals)-1 {
				t += ","
			}
			b.word(t, false)
		}
	}
	b.flush(false)
	b.start(NewsBoxWidth, NewsLeft, newsYellowTop)
	b.word("GELBE KARTEN: ", true)
	if len(rep.yellow) == 0 {
		b.word("KEINE", true)
	}
	for i, y := range rep.yellow {
		if i < len(rep.yellow)-1 {
			y += ","
		}
		b.word(y, true)
	}
	b.flush(true)
	b.start(NewsBoxWidth, NewsLeft, newsRedTop)
	b.word("ROTE KARTEN  : ", true)
	if rep.red == "" {
		b.word("KEINE", true)
	} else {
		b.word(rep.red, true)
	}
	b.flush(true)
	np.Box = b.lines
	return np
}

// newsReport rebuilds the match report record of manager mi from the
// events of the match day, like f_1a58_20b2 and f_2e3a_223e write it.
func (md *MatchDay) newsReport(mi int) *newsReport {
	s := md.s
	hm := &md.humans[mi]
	f := md.Fixtures[hm.fixture]
	rep := &newsReport{own: f.Home, opp: f.Away}
	if hm.side == 1 {
		rep.own, rep.opp = f.Away, f.Home
	} else {
		rep.attendance = f.Attendance
		m := s.Managers[mi]
		// The original compares the capacity with the season's sum of
		// the league attendances (quirk: true almost only on the first
		// home match).
		rep.soldOut = m.Capacity() == m.SeasonAudience
	}
	for _, e := range md.Events {
		if e.Fixture != hm.fixture {
			continue
		}
		switch e.Kind {
		case EvGoal, EvMiss:
			c := newsChance{minute: e.Minute, opp: e.Side != hm.side}
			if !c.opp {
				c.name = e.Player
			}
			rep.chances++
			if e.Kind == EvMiss {
				rep.misses = append(rep.misses, c)
				continue
			}
			if c.opp {
				rep.ga++
			} else {
				rep.gf++
			}
			rep.diff = rep.gf - rep.ga
			switch {
			case rep.diff > 0:
				rep.state = 1
			case rep.diff < 0:
				rep.state = 2
			default:
				rep.state = 0
			}
			if rep.diff < rep.minDiff {
				rep.minDiff = rep.diff
				rep.minScore = [2]int{rep.gf, rep.ga}
			}
			if rep.maxDiff < rep.diff {
				rep.maxDiff = rep.diff
				rep.maxScore = [2]int{rep.gf, rep.ga}
			}
			rep.goals = append(rep.goals, c)
		case EvYellow:
			if e.Manager == mi {
				rep.yellow = append(rep.yellow, e.Player)
			}
		case EvYellowRed, EvRed:
			if e.Manager == mi {
				rep.red = e.Player
			}
		}
	}
	return rep
}

// newsExpander expands the templates of the newspaper (f_2e3a_0964):
// %a/%b own/opponent club, %c/%d best/worst player, %t manager, %0:%1 the
// result, %2:%3 the score at the largest deficit, %4:%5 the score at the
// largest lead, %9 attendance, %e end of the first headline line and
// %xDN alt1#alt2#...#% a choice: alternative rnd(1,N) for D=0, else the
// one of selector D.
type newsExpander struct {
	md  *MatchDay
	mi  int
	rep *newsReport
}

func (e *newsExpander) slotName(slot int) string {
	if slot <= 0 {
		return ""
	}
	s := e.md.s
	return s.Persons[s.Squads[e.mi][slot-1].Person].Name
}

// expand expands t from p up to '#', the end, or %e; it returns the text
// and the position after it.
func (e *newsExpander) expand(t string, p int) (string, int) {
	var out strings.Builder
	p = e.expandInto(&out, t, p)
	return out.String(), p
}

func (e *newsExpander) expandInto(out *strings.Builder, t string, p int) int {
	at := func(i int) byte {
		if i < 0 || i >= len(t) {
			return 0
		}
		return t[i]
	}
	s := e.md.s
	rep := e.rep
	for {
		c := at(p)
		if c == '#' || c == 0 {
			return p
		}
		if c != '%' {
			out.WriteByte(c)
			p++
			continue
		}
		p++
		end := false
		switch at(p) {
		case 'x':
			p++
			sel := int(at(p)) - '0'
			p++
			var k int
			if sel == 0 {
				k = s.Rand.Between(1, int(at(p))-'0')
				p++
			} else {
				p++
				k = 1
				if sel > 0 && sel < len(rep.sel) {
					k = rep.sel[sel]
				}
			}
			// The original runs off into memory for a selector 0 (the
			// broken "%xDE" of one headline reads an unknown byte).
			k = max(k, 1)
			for {
				if k <= 1 {
					p = e.expandInto(out, t, p)
				} else {
					for {
						if at(p) == '%' && at(p+1) == 'x' {
							p += 4
							for at(p) != '%' && at(p) != 0 {
								p++
							}
						}
						p++
						if at(p) == '#' || at(p) == 0 {
							break
						}
					}
				}
				p++
				k--
				if k == 0 || p >= len(t) {
					break
				}
			}
			for p < len(t) && !(at(p) == '%' && at(p-1) == '#') {
				if at(p) == '%' && at(p+1) == 'x' {
					p += 4
					for at(p) != '%' && at(p) != 0 {
						p++
					}
				}
				p++
			}
		case '0':
			out.WriteString(strconv.Itoa(rep.gf))
		case '1':
			out.WriteString(strconv.Itoa(rep.ga))
		case '2':
			out.WriteString(strconv.Itoa(rep.minScore[0]))
		case '3':
			out.WriteString(strconv.Itoa(rep.minScore[1]))
		case '4':
			out.WriteString(strconv.Itoa(rep.maxScore[0]))
		case '5':
			out.WriteString(strconv.Itoa(rep.maxScore[1]))
		case '9':
			out.WriteString(strconv.FormatInt(rep.attendance, 10))
		case 'a':
			out.WriteString(s.Teams[rep.own].Name)
		case 'b':
			out.WriteString(s.Teams[rep.opp].Name)
		case 'c':
			out.WriteString(e.slotName(rep.hero))
		case 'd':
			out.WriteString(e.slotName(rep.loser))
		case 't':
			out.WriteString(s.Managers[e.mi].Name)
		case 'e':
			end = true
		}
		p++
		if end {
			return p
		}
	}
}

// newsWriter breaks words into justified lines (f_2e3a_06df): a line that
// would get too wide is widened to the full width by inserting spaces at
// the gaps between its words, one gap after the other.
type newsWriter struct {
	width func(string) int
	w     int // width of the column
	x, y  int
	line  string
	pos   []int // end of each word in line
	lines []NewsLine
}

func (w *newsWriter) start(width, x, y int) {
	w.w, w.x, w.y = width, x, y
	w.line, w.pos = "", nil
}

func (w *newsWriter) emit(split bool) {
	if w.line != "" {
		w.lines = append(w.lines, NewsLine{X: w.x, Y: w.y, Text: w.line, Split: split && w.w == NewsBoxWidth})
	}
}

// word adds a word; split tells how a line completed by it is drawn. It
// returns the baseline of the current line.
func (w *newsWriter) word(s string, split bool) int {
	cand := w.line
	if cand != "" {
		cand += " "
	}
	cand += s
	if w.width(cand) > w.w {
		n := len(w.pos)
		if n > 0 {
			n--
		}
		i := 0
		for guard := 0; w.width(w.line) < w.w && guard < 400; guard++ {
			p := len(w.line)
			if i < len(w.pos) {
				p = w.pos[i]
			}
			i++
			w.line = w.line[:p] + " " + w.line[p:]
			for j := i; j < n; j++ {
				w.pos[j]++
			}
			if n <= i {
				i = 0
			}
		}
		w.emit(split)
		w.y += NewsLineStep
		w.line, w.pos = "", nil
	}
	if w.line != "" {
		w.line += " "
	}
	w.line += s
	w.pos = append(w.pos, len(w.line))
	return w.y
}

// flush draws the rest of the text without justification; a trailing comma
// is dropped.
func (w *newsWriter) flush(split bool) {
	w.line = strings.TrimSuffix(w.line, ",")
	w.emit(split)
}
