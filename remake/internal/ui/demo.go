package ui

// Demo returns a scene filled with sample data, for screenshots during
// development (bmp -demo NAME -shot FILE).
func Demo(name string) Scene {
	switch name {
	case "mainmenu":
		m := &MainMenu{
			Header: func() Header {
				return Header{Matchday: "1. Spieltag", Manager: "TEST", Logo: 89,
					Weekday: "Samstag", Day: 29, Month: "Juli", Year: 1992}
			},
			Left:  [3]MenuGroup{{Icon: IconDesk}, {Icon: IconClub}, {Icon: IconTeam}},
			Right: [3]MenuGroup{{Icon: IconCompetitions}, {Icon: IconDisk}, {Icon: IconContinue}},
		}
		nop := func(*App) {}
		m.Left[0].Items = [9]MenuItem{{IconFinanceChart, nop}, {IconHistory, nop}, {IconTrophies, nop},
			{IconStadium, nop}, {IconBank, nop}, {IconSponsors, nop}, {}, {}, {}}
		m.open = &m.Left[0]
		return m
	}
	return NewTitle()
}
