# v-code

Voice coding trajectories: real multi-turn voice sessions between a developer and a coding agent, one folder
per task. In each folder, every commit is one turn, so you can step through the history to see what was
said, what the agent did, and where it went wrong.

| Task | Agent | Axes | Turns | Hidden tests |
|---|---|---|---|---|
| [expr-try-catch-errors](expr-try-catch-errors/) | Claude Code (claude-opus-5-5) | V6 + V8 (Italian) + V2 | 9 | 72 of 74 |

To see one task's turns: `git log --reverse -- <task>/` and `git show <commit> -- <task>/repo`.
