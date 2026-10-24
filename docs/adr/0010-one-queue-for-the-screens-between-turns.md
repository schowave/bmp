---
status: accepted
---

# One queue for the screens between the turns

The original shows every dialog and screen where its rule runs: the game
loop `f_1a58_3176` calls the trophy screen (`f_17c2_274d` @3945), the flag
SAISONENDE (`f_1a58_4597` @3c45), the season report (`f_0cb5_0012` @43b5)
and the cup draws (`f_17c2_0006`) one after another. The rules of the remake
draw nothing (`remake/internal/game`), so they queue what the screens have to
show. They had three queues: the flags (`State.Flags`), the cup draws
(`State.Draws`) and the notices (alerts, cards, trophies, season reports,
popups). The driver of the UI showed them by a fixed priority (flags, notices,
draws), which loses the order between them. When the last manager became
champion, SAISONENDE came before the trophy screen; in the original it comes
after.

## Considered Options

- One queue, `State.Notices`: the flag and the draw become kinds of notice
  (`NoticeFlag`, `NoticeDraw`), the rules queue in the order of the original,
  and the driver shows the queue in order. Chosen.
- Keep the three queues and special-case the order in the driver: rejected,
  every new case needs another rule in the UI that the rules' tests cannot
  see.
- Show screens from the rules through callbacks into the UI: rejected,
  `game` would wait for the player and could no longer play seasons in tests.

## Consequences

- The order the player sees is decided by `remake/internal/game` and tested
  there (`TestSeasonEndOrder`); the driver only shows the queue.
- The queue is a field of `State` and goes into saved games (it is empty at
  the turns, where the game is saved). Games saved before keep loading; a
  flag or draw still in their old fields is dropped.
- The high-score entries (`State.Highscores`) stay apart: at a season end the
  original updates the table without showing it (`f_322b_2366(2)`).
- The messages of the panel (`Manager.Inbox`) stay apart: they are read in
  the main menu, not shown between the turns.
