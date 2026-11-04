# v-code

Voice coding trajectories: real multi-turn voice sessions between a developer and a coding agent, one folder
per task. In each folder, every commit is one turn, so you can step through the history to see what was
said, what the agent did, and where it went wrong.

| Task | Agent | Axes | Turns | Hidden tests |
|---|---|---|---|---|
| [expr-try-catch-errors](expr-try-catch-errors/) | Claude Code (claude-opus-5-5) | V6 + V8 (Italian) + V2 | 9 | 72 of 74 |
| [adaptix-name-mapping-aliases](adaptix-name-mapping-aliases/) | Claude Code (claude-opus-5-5) | V3 + V2 | 6 | 44 of 44 |
| [adaptix-name-mapping-aliases-codex](adaptix-name-mapping-aliases-codex/) | Codex CLI (gpt-6.1-sol), realtime /voice | V3 + V2 | 7 | 44 of 44 |
| [anko-default-function-arguments](anko-default-function-arguments/) | Claude Code (claude-opus-5-5) | V6 + V2 | 6 | 124 of 124 |

Each task folder has a README (task, agent, axes, how to read it), `turns/turn-NN.md` (what was said, what the
agent received, dictation errors, failure points, agent replies and tool calls), `repo/` (the code after each
turn) and `grading.md`.

To see one task's turns: `git log --reverse -- <task>/` and `git show <commit> -- <task>/repo`.
