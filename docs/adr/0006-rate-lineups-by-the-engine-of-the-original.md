---
status: accepted
---

# Rate lineups and training by the engine of the original

The remake's hidden keys `o` set the best lineup, training plan and ticket
price. "Best" needs a rating. The original has no such rating; its rules only
say what a lineup does in a match.

## Considered Options

- The expected points of the next match (2 a win, 1 a draw), from the
  original's own calculation: the lineup strengths (`f_0f9d_0002`), the
  chances of a half (`f_0f9d_08e9`) and the goal odds (`f_0f9d_0c3b`), against
  the next opponent in his venue; without one, the mean over the clubs of the
  league, home and away. Chosen (`internal/game/lineupmodel.go`); tests check
  it against runs of the engine's dice.
- A sum of the strengths of the lines: rejected. It is not what the engine
  plays, and the search used the lines' balance bonus to pile the team into
  the midfield.
- The expected goal difference against one average club of the league:
  rejected. The goal odds have a step (attack 0 to 3 points behind the defence
  scores for sure, 1 point ahead two in three); against one club the search
  aimed at that step and gained nothing against the others. The mean over the
  league smooths it, and against the known next opponent the step is real.
- The goal difference instead of points: rejected. It weighs a goal more
  scored the same as one less let in, which a result does not.
- The training plan by an analytic expectation of `develop`: rejected for
  plays of the rest of the season on copies of the game with the original's
  training, injuries in training and match fatigue (`trainingopt.go`): the
  thresholds of `develop` make the expectation complicated, the copies are
  cheap (half a second a press) and use the rules as they are.

## Consequences

- The rating is an expectation with the dice of the lineup values replaced by
  their mean (league) or the mean of eight fixed series (next opponent); a
  change must gain 0,03 points, as the model is no more exact.
- Cards, injuries and substitutions of the match are left out. So the EINSATZ
  slider is not optimised: its strength gain is in the rating, its cards are
  not, and simulated seasons showed no gain from it.
- Everything the keys use is the engine; when a rule of the engine is ported
  more exactly, the keys follow without a change of their own.
