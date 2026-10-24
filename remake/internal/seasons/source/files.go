package source

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// The raw data of a season in remake/data/seasons/YYYY-YY: CSV files that
// can be read and corrected by hand, and the lists of articles read and
// problems found.
const (
	tablesFile   = "tables.csv"
	squadsFile   = "squads.csv"
	logosFile    = "logos.csv"
	europeFile   = "europe.csv"
	articlesFile = "articles.txt"
	missingFile  = "missing.txt"
)

var posCodes = []string{"", "TW", "ABW", "MIT", "ANG"}

func writeCSV(path string, header []string, rows [][]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	if err := w.Write(header); err != nil {
		_ = f.Close()
		return err
	}
	if err := w.WriteAll(rows); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// ReadCSV reads the CSV file path of fsys without its header line.
func ReadCSV(fsys fs.FS, path string) ([][]string, error) {
	f, err := fsys.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[1:], nil
}

// WriteSeason writes s into dir.
func WriteSeason(dir string, s *Season) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var rows [][]string
	for _, r := range s.Tables {
		rows = append(rows, []string{strconv.Itoa(r.Tier), r.Group, strconv.Itoa(r.Rank), r.Club, r.Label,
			strconv.Itoa(r.Won), strconv.Itoa(r.Drawn), strconv.Itoa(r.Lost), strconv.Itoa(r.GoalsFor), strconv.Itoa(r.GoalsAgainst)})
	}
	if err := writeCSV(filepath.Join(dir, tablesFile),
		[]string{"tier", "group", "rank", "club", "label", "won", "drawn", "lost", "goals_for", "goals_against"}, rows); err != nil {
		return err
	}
	rows = nil
	for _, club := range sortedKeys(s.Squads) {
		for _, p := range s.Squads[club] {
			since := ""
			if p.Since > 0 {
				since = strconv.Itoa(p.Since)
			}
			rows = append(rows, []string{club, p.Name, p.Surname, posCodes[p.Pos], strconv.Itoa(p.Games), since, p.Source})
		}
	}
	if err := writeCSV(filepath.Join(dir, squadsFile), []string{"club", "name", "surname", "position", "games", "since", "source"}, rows); err != nil {
		return err
	}
	rows = nil
	for _, club := range sortedKeys(s.Logos) {
		rows = append(rows, []string{club, s.Logos[club], s.Colors[club]})
	}
	if err := writeCSV(filepath.Join(dir, logosFile), []string{"club", "file", "colors"}, rows); err != nil {
		return err
	}
	rows = nil
	for _, e := range s.Europe {
		rows = append(rows, []string{strconv.Itoa(e.Cup), e.Club, e.Label, e.Country, strconv.Itoa(e.Mentions)})
	}
	if err := writeCSV(filepath.Join(dir, europeFile), []string{"cup", "club", "label", "country", "mentions"}, rows); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, articlesFile), []byte(strings.Join(s.Articles, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, missingFile), []byte(strings.Join(s.Missing, "\n")+"\n"), 0o644)
}

// TMFile is the file of the Transfermarkt squads of season id in the raw
// data: kept apart from the openly licensed data of the seasons.
func TMFile(id string) string { return path.Join("transfermarkt", id+".csv") }

// WriteTM writes the Transfermarkt squads of a season into file: per club
// (its article) the name Transfermarkt gives it and the players in their
// order.
func WriteTM(file string, squads map[string][]Player, names map[string]string) error {
	var rows [][]string
	for _, club := range sortedKeys(squads) {
		for _, p := range squads[club] {
			rows = append(rows, []string{club, names[club], p.Name, posCodes[p.Pos]})
		}
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return writeCSV(file, []string{"club", "tm_name", "name", "position"}, rows)
}

// ReadTMSquads reads the Transfermarkt squads of file in fsys by club; none
// when there is no file.
func ReadTMSquads(fsys fs.FS, file string) (map[string][]Player, error) {
	out := map[string][]Player{}
	rows, err := ReadCSV(fsys, file)
	if errors.Is(err, fs.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for i, r := range rows {
		if len(r) != 4 {
			return nil, fmt.Errorf("%s line %d: %d fields", file, i+2, len(r))
		}
		out[r[0]] = append(out[r[0]], Player{Name: r[2], Surname: r[2], Pos: posOf(r[3]), Games: -1, Source: "transfermarkt"})
	}
	return out, nil
}

// ReadSeason reads the season of year from fsys (the directory of the
// season).
func ReadSeason(fsys fs.FS, year int) (*Season, error) {
	s := &Season{Year: year, Squads: map[string][]Player{}, Logos: map[string]string{}, Colors: map[string]string{}}
	atoi := func(v string) int { n, _ := strconv.Atoi(strings.TrimSpace(v)); return n }
	rows, err := ReadCSV(fsys, tablesFile)
	if err != nil {
		return nil, err
	}
	for i, r := range rows {
		if len(r) != 10 {
			return nil, fmt.Errorf("%s line %d: %d fields", tablesFile, i+2, len(r))
		}
		s.Tables = append(s.Tables, TierRow{Tier: atoi(r[0]), Row: Row{Group: r[1], Rank: atoi(r[2]), Club: r[3], Label: r[4],
			Won: atoi(r[5]), Drawn: atoi(r[6]), Lost: atoi(r[7]), GoalsFor: atoi(r[8]), GoalsAgainst: atoi(r[9])}})
	}
	if rows, err = ReadCSV(fsys, squadsFile); err != nil {
		return nil, err
	}
	for i, r := range rows {
		if len(r) != 7 {
			return nil, fmt.Errorf("%s line %d: %d fields", squadsFile, i+2, len(r))
		}
		s.Squads[r[0]] = append(s.Squads[r[0]], Player{Name: r[1], Surname: r[2], Pos: posOf(r[3]), Games: atoi(r[4]), Since: atoi(r[5]), Source: r[6]})
	}
	if rows, err = ReadCSV(fsys, logosFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	for i, r := range rows {
		if len(r) != 3 {
			return nil, fmt.Errorf("%s line %d: %d fields", logosFile, i+2, len(r))
		}
		s.Logos[r[0]], s.Colors[r[0]] = r[1], r[2]
	}
	if rows, err = ReadCSV(fsys, europeFile); err != nil {
		return nil, err
	}
	for i, r := range rows {
		if len(r) != 5 {
			return nil, fmt.Errorf("%s line %d: %d fields", europeFile, i+2, len(r))
		}
		s.Europe = append(s.Europe, EuroClub{Cup: atoi(r[0]), Club: r[1], Label: r[2], Country: r[3], Mentions: atoi(r[4])})
	}
	return s, nil
}

// posOf reads a position code (TW, ABW, MIT, ANG).
func posOf(code string) int {
	for k, c := range posCodes {
		if c == code {
			return k
		}
	}
	return PosNone
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
