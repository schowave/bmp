package source

import (
	"errors"
	"strings"
)

// ClubInfo is what the infobox of a club article says.
type ClubInfo struct {
	Logo   string // "Datei:..." of the current logo
	Colors string // "Schwarz-Gelb", "Rot und Weiß"
}

// ClubInfobox reads the infobox of the club article title.
func (c *Client) ClubInfobox(title string) (ClubInfo, error) {
	text, err := c.Raw(title)
	if errors.Is(err, ErrNotFound) {
		return ClubInfo{}, nil
	}
	if err != nil {
		return ClubInfo{}, err
	}
	i := strings.Index(text, "{{Infobox Fußball")
	if i < 0 {
		return ClubInfo{}, nil
	}
	body, ok := templateBody(text[i:])
	if !ok {
		return ClubInfo{}, nil
	}
	p := params(body)
	var info ClubInfo
	for _, k := range []string{"image", "bild", "logo", "wappen"} {
		if m := linkRE.FindStringSubmatch(p[k]); m != nil && isFileLink(strings.TrimSpace(m[1])) {
			info.Logo = "Datei:" + strings.TrimSpace(strings.SplitN(m[1], ":", 2)[1])
			break
		}
	}
	for _, k := range []string{"farben_verein", "vereinsfarben", "farben"} {
		if v := plainText(p[k]); v != "" {
			info.Colors = v
			break
		}
	}
	return info, nil
}
