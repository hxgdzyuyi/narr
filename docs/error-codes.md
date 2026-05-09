# narrc Error Codes

Diagnostics are stable enough for tests and CI. Codes use `E` for errors and `W` for warnings.

## Ranges

| Range | Area |
| --- | --- |
| `E0000-E0013` | project discovery, config, and file collection |
| `E0200-E0222` | parser and source-file shape |
| `E0301-E0313` | symbols, imports, references, and query environments |
| `E0401-E0418` | model construction, schema, and type checking |
| `E0501-E0513` | state values, anchors, effects, and timeline queries |
| `E0601-E0627` | structure checks and derived narrative views |
| `E0701-E0726` | query expression evaluation |
| `E0801-E0807` | build, test, and command-specific execution |
| `E0901-E0905` | test-file semantic validation |
| `W0001-W0002` | project configuration warnings |

## Project And Config

| Code | Meaning |
| --- | --- |
| `E0000` | current working directory could not be determined |
| `E0001` | project root could not be discovered or does not contain `narr.toml` |
| `E0002` | `narr.toml` could not be read |
| `E0003` | `narr.toml` could not be parsed |
| `E0004` | required `[project]` table is missing |
| `E0005` | `[project]` is not a TOML table |
| `E0006` | unsupported top-level config key is present |
| `E0007` | unsupported `[project]` field is present |
| `E0008` | `[project].name` is missing |
| `E0009` | `[project].version` is missing |
| `E0010` | project version is incompatible with the compiler |
| `E0011` | project path could not be read while collecting files |
| `E0012` | project-relative file path could not be computed |
| `E0013` | project scan failed |
| `W0001` | unknown top-level config key was ignored |
| `W0002` | unknown `[project]` field was ignored |

## Parser

| Code | Meaning |
| --- | --- |
| `E0200` | source file could not be read |
| `E0201` | expression has trailing tokens |
| `E0202` | file is missing a namespace declaration |
| `E0203` | `.test.narr` contains a non-test top-level declaration |
| `E0204` | `.narr` contains a test declaration |
| `E0205` | unexpected top-level token |
| `E0206` | expected an identifier |
| `E0207` | expected an identifier after `.` |
| `E0208` | expected a token such as `{`, `}`, `:`, or `)` |
| `E0209` | expected a specific keyword |
| `E0210` | expected a named identifier |
| `E0211` | expected a field or block statement |
| `E0212` | expected a field body |
| `E0213` | expected a field value after `:` |
| `E0214` | expected a field statement |
| `E0215` | expected a block item |
| `E0216` | unknown top-level declaration keyword |
| `E0217` | expected an effect statement |
| `E0218` | expected an assignment operator |
| `E0219` | expected `in` after `not` |
| `E0220` | expected an expression |
| `E0221` | unknown binder domain type |
| `E0222` | expected a test statement |

## Resolve

| Code | Meaning |
| --- | --- |
| `E0301` | duplicate declaration |
| `E0302` | duplicate chapter code |
| `E0303` | duplicate volume code |
| `E0304` | invalid volume code in declaration |
| `E0305` | invalid chapter code in declaration |
| `E0306` | imported namespace does not exist |
| `E0307` | import alias conflicts with another import |
| `E0308` | query namespace cannot be determined |
| `E0309` | query namespace does not exist |
| `E0310` | query namespace import alias is ambiguous |
| `E0311` | unknown reference |
| `E0312` | chapter alias is ambiguous |
| `E0313` | volume alias is ambiguous |

## Model And Type Checking

| Code | Meaning |
| --- | --- |
| `E0401` | invalid volume code while building the model |
| `E0402` | invalid chapter code while building the model |
| `E0404` | entity or state target field does not exist |
| `E0405` | set operator used on a non-set field |
| `E0406` | list append used on a non-list field |
| `E0407` | effect target does not include entity and field |
| `E0408` | effect target does not resolve to an entity |
| `E0409` | value is not assignable to the target type |
| `E0410` | enum value is not a bare member |
| `E0411` | enum member does not exist |
| `E0412` | reference value does not resolve |
| `E0413` | invalid type expression |
| `E0414` | unknown type |
| `E0415` | declaration field is not supported by that declaration kind |
| `E0416` | declaration field has the wrong value or block shape |
| `E0417` | declaration field has an invalid enumerated value |
| `E0418` | novel length block contains an invalid length statement |

## State

| Code | Meaning |
| --- | --- |
| `E0501` | invalid integer in state expression |
| `E0502` | effect target does not resolve to a state field |
| `E0503` | expected a `state(...)` expression |
| `E0504` | unknown state checkpoint |
| `E0505` | missing anchor |
| `E0506` | anchor must be a reference |
| `E0507` | invalid chapter anchor code |
| `E0508` | chapter anchor suffix is unsupported |
| `E0509` | beat anchor suffix is unsupported |
| `E0510` | anchor reference is not a chapter or beat |
| `E0511` | invalid volume anchor code |
| `E0512` | volume anchor suffix is unsupported |
| `E0513` | volume contains no chapters |

## Structure

| Code | Meaning |
| --- | --- |
| `E0601` | beat appears more than once in a chapter beats list |
| `E0602` | beat is missing a chapter anchor |
| `E0603` | beat anchor does not point to a chapter |
| `E0604` | beat is not listed in its anchored chapter |
| `E0605` | beat anchor and chapter beats list disagree |
| `E0606` | beat assigns conflicting values to one state field |
| `E0607` | beat both adds and removes the same set value |
| `E0608` | start pattern points to a missing state checkpoint |
| `E0609` | start pattern is missing `at` |
| `E0610` | start target kind is not valid |
| `E0611` | promise is missing `setup_at` |
| `E0612` | promise `payoff_at` is before setup |
| `E0613` | promise `payoff_by` is before setup |
| `E0614` | thread is missing `starts_at` |
| `E0615` | arc is missing `subject` |
| `E0616` | arc is missing `starts_at` |
| `E0617` | arc is missing `state_field` |
| `E0618` | arc state field does not exist on subject |
| `E0619` | arc initial state is outside declared states |
| `E0620` | arc state change is outside declared states |
| `E0621` | hidden invariant is revealed before its allowed anchor |
| `E0622` | start pattern anchor does not match target start anchor |
| `E0623` | condition, precondition, or invariant assertion failed |
| `E0624` | reference has the wrong symbol kind |
| `E0625` | reference has the wrong required symbol kind |
| `E0626` | expected a reference expression |
| `E0627` | beat is listed in more than one chapter |

## Query, Build, And Test

| Code | Meaning |
| --- | --- |
| `E0701` | invalid integer during query evaluation |
| `E0702` | unsupported expression kind |
| `E0703` | unsupported postfix operator |
| `E0704` | unsupported binary operator |
| `E0705` | non-reference value has no property |
| `E0706` | reference does not support the requested anchor suffix |
| `E0708` | `chapters_in` argument is not a volume |
| `E0709` | `beats` argument is not a chapter |
| `E0710` | `build` argument is not a chapter |
| `E0711` | unknown function |
| `E0712` | view function argument is not a chapter |
| `E0713` | view function chapter does not exist |
| `E0714` | binder source is not a collection |
| `E0715` | invalid `state(...)` query |
| `E0716` | unknown collection |
| `E0717` | `volume_of` argument is invalid |
| `E0718` | `chapter_of` argument is invalid |
| `E0719` | `previous` or `next` argument is invalid |
| `E0720` | `chapter_distance` arguments are invalid |
| `E0721` | `chapters_between` arguments are invalid |
| `E0722` | unknown builtin |
| `E0723` | function arity mismatch |
| `E0724` | unknown chapter |
| `E0725` | object property does not exist |
| `E0726` | build object chapter does not exist |
| `E0801` | build target chapter is unknown |
| `E0802` | build target has no structure view |
| `E0803` | build target has no state checkpoint |
| `E0804` | build target resolved to a non-chapter |
| `E0805` | build target chapter code is invalid |
| `E0806` | build JSON could not be written |
| `E0807` | test file selector matched no `.test.narr` file |
| `E0901` | duplicate local test binding |
| `E0902` | test tags are not a set |
| `E0903` | test condition or assert expression is not boolean |
| `E0904` | test query expected a collection with matching element type |
| `E0905` | narrative predicate target type is invalid |
