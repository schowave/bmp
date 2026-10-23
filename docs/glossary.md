# Glossary

The terms of *Bundesliga Manager Professional* as the specs in `docs/re/` and
the code of `remake/internal/game` use them. The German word is the one the
game shows; code and specs use the English one.

## Language

**Manager**:
A human player of the game; up to four (index 0..3), each running one club.
_Avoid_: player (that is a footballer), user

**Club** (Verein):
One of the 200 entries of the team table (id 0..199); leagues are id ranges of
it, and promotion and relegation swap identities between ids.
_Avoid_: team when the id is meant

**League** (Liga):
1. Liga (18 clubs, 34 matchdays), 2. Liga (20, 38) or Amateur-Oberliga (20, 38),
index 0..2. The remake names the leagues after the season: the second
Regionalliga until 1973/74; the third Amateurliga until 1977/78,
Regionalliga from 1994/95, 3. Liga from 2008/09.

**Real season** (Saison):
A season from 1990/91 on whose clubs, players and logos the remake builds
in (`remake/internal/seasons`), chosen before the setup; as opposed to the
clubs of the original.

**Matchday** (Spieltag):
One round of league matches; each league counts its own.

**Turn day**:
A day of the calendar on which the managers act, two per week; the season ends
after day 322.
_Avoid_: round, tick

**Cup** (Pokal):
DfB-Pokal, Pokal der Landesmeister, Pokal der Pokalsieger or UEFA-Pokal, in
that order (index 0..3); the last three are the Europapokal.

**Relegation**:
The two-leg play-off between the 1. Liga and the 2. Liga at the end of the
season, for the last place in the 1. Liga.

**Nachholspiel**:
A league match postponed in winter and played later.

**Person**:
An entry of the person table: a footballer, with name, age and position;
person 0 marks an empty squad slot.

**Squad** (Kader):
The 24 slots of a manager's players; a slot holds the contract and the season
figures of one person.

**Transfer market** (Transfermarkt):
The players for sale, kept as the squad of a pseudo manager with index 4 and
12 slots.

**Lineup** (Aufstellung):
Slot positions 1..11 are the team, 12 and above the reserve, 0 not named.

**Skills**:
Kondition, Technik and Form of a slot; with them the strength (Stärke) of a
player.

**Tendency** (Tendenz):
Whether a player's form is falling, constant or rising (value 30..68).

**Erschöpfung**:
Fatigue of a player, 50..150, normally 60..100.

**Loan** (Leihe):
A player borrowed from another club; the slot records the club he came from.

**Level**:
Difficulty 0..5 as the options show it; the code stores `L = 5 - Level`, so
`L = 5` is the easiest.

**Flag** (Fahne):
The German flag over the whole screen with a word on it (`f_0310_01f2`): the
competition before the matches of a match day, WINTERPAUSE, SAISONENDE.
_Avoid_: banner

**Optimal lineup**:
The lineup the remake's hidden key `o` sets: the most expected points against
the next opponent, or against the league without one.

**Training plan**:
The settings of the training screen: the 20 training points, the 10 position
points and the intensity. The remake's hidden key `o` sets the one with the
strongest eleven over the rest of the season.

**Ticket price** (Eintrittspreis):
The price a manager asks at his home matches, 5..25 DM; with the attendance it
gives the gate money. The remake's hidden key `o` on the stadium screen sets
the one with the most gate money at the next home match.

**Timed message**:
A message of the panel with a lifetime (an injury in training, riot damage,
the comfort downgrade, a finished construction, a retirement announcement):
it goes after three turns; at most ten wait per manager.

**Original**:
The DOS game, `BMMAIN.EXE` in `bmp/`; its functions are named `f_SSSS_OOOO`
after their address in the unpacked image.
_Avoid_: legacy, old version
