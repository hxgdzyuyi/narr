# Parser EBNF Coverage

Source of truth: `docs/syntax.md` v0.3.5.

Status values:

- `done`: parser has an explicit function or function group and positive tests.
- `partial`: syntax is parsed structurally for M2, but later semantic stages still need checks.

| syntax.md section | EBNF production | parser function | positive tests | negative tests | status |
| --- | --- | --- | --- | --- | --- |
| 1 | `source_file` | `ParseNarrFile`, `ParseTestFile` | `TestParseExamplesProject` | `TestTestFileRejectsNarrDeclarations` | done |
| 1, 21 | `narr_file`, `test_file` | `parseFile` | `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 3 | `narr_declaration` | `parseDecl` | `TestParseNarrDeclarations` | `TestTestFileRejectsNarrDeclarations` | done |
| 4 | `namespace_decl`, `namespace_path` | `parseNamespacePath`, `parseDottedName` | `TestParseNarrDeclarations`, `TestParseTestFile` | `TestParseErrorsContainLineAndColumn` | done |
| 4 | `import_decl`, `alias_clause` | `parseImportAfterKeyword`, `parseAlias` | `TestParseNarrDeclarations`, `TestParseTestFile` | `TestParseErrorsContainLineAndColumn` | done |
| 5 | `identifier`, `integer`, `string`, `text`, `bool` | `lexer.Next`, `parseValueExpr`, `parseTextValue` | `TestLexerUnicodeIdentifiersAndOperators`, `TestParseNarrDeclarations` | `TestParseErrorsContainLineAndColumn` | done |
| 5 | `language_tag` | `parseLanguageTag` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 5 | `qualified_ref` | `parseDottedName`, `parsePathExpr`, `parseRef` | `TestParseNarrDeclarations`, `TestParseTestFile` | `TestParseErrorsContainLineAndColumn` | done |
| 5 | `literal`, `list_expr`, `set_expr`, `expr_list` | `parseValueExpr`, `parseDelimitedExprList` | `TestParseNarrDeclarations`, `TestParseTestFile` | `TestParseErrorsContainLineAndColumn` | done |
| 6 | `chapter_code`, `volume_code`, `chapter_part_code`, `code_segment`, `unsigned_integer` | `parseDottedName` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | partial |
| 7 | `novel_decl`, `novel_field` | `parseDecl`, `parseBlock`, `parseNamedField` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 7 | `length_block`, `length_stmt`, `length_value`, `length_unit` | `parseLengthBlock`, `isLengthUnitOnSameLine` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 8 | `volume_decl`, `volume_field` | `parseDecl`, `parseAlias`, `parseBlock` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 9 | `chapter_decl`, `chapter_field` | `parseDecl`, `parseAlias`, `parseBlock` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 10 | `beat_decl`, `beat_anchor`, `beat_field`, `narr_link_expr` | `parseDecl`, `parseAnchorRef`, `parseBlock` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 11 | `effect_block`, `effect_stmt`, `assignment`, `set_add`, `set_remove`, `list_append` | `parseEffectBlock`, `parseEffectLikeStmt` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 12 | `time_ref`, `anchor_ref`, `anchor_suffix` | `parseAnchorRef`, `parseRef`, `parsePathExpr` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | partial |
| 13 | `start_pattern_decl`, `start_pattern_field`, `start_target_block`, `start_target` | `parseDecl`, `parseNamedField`, `parseStartTargetBlock` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 14 | `promise_decl`, `promise_field` | `parseDecl`, `parseBlock` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 15 | `thread_decl`, `thread_field` | `parseDecl`, `parseBlock` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 16 | `arc_decl`, `arc_field` | `parseDecl`, `parseBlock` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 17 | `invariant_decl`, `invariant_field`, `hidden_rule`, `always_rule` | `parseDecl`, `parseNamedField`, `parseConditionBlock` | `TestParseNarrDeclarations` | `TestParseErrorsContainLineAndColumn` | done |
| 18 | `enum_decl`, `class_decl`, `class_field`, `default_clause`, `type_expr` | `parseDecl`, `parseEnumBlock`, `parseClassField`, `parseTypeExpr` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 18 | `place_decl`, `character_decl`, `collective_decl`, `faction_decl`, `object_decl`, `fact_decl` | `parseDecl`, `parseOptionalBlock`, `parseTextValue` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 18 | `in_clause`, `class_clause` | `parseDecl`, `parseRef` | `TestParseNarrDeclarations`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 21 | `test_decl`, `test_attr`, `tags_clause` | `parseTestDecl` | `TestParseTestFile`, `TestParseExamplesProject` | `TestTestFileRejectsNarrDeclarations` | done |
| 22 | `test_stmt`, `assert_stmt`, `message_clause` | `parseTestStmt`, `parseAssertStmt` | `TestParseTestFile`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 22 | `let_stmt`, `forall_stmt`, `exists_stmt` | `parseLetStmt`, `parseForallStmt`, `parseExistsStmt` | `TestParseTestFile`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 22 | `binder`, `in_query_clause`, `where_clause` | `parseBinder` | `TestParseTestFile`, `TestParseExpressionPrecedence` | `TestParseErrorsContainLineAndColumn` | done |
| 23 | `domain_type` | `parseBinder`, `isDomainType` | `TestParseTestFile`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 24 | `query_expr`, `collection_query`, `relation_query`, `projection_query` | `parseNamedValueExpr`, `parseCollectExpr` | `TestParseTestFile`, `TestParseExpressionPrecedence` | `TestParseErrorsContainLineAndColumn` | done |
| 25 | `expr`, `implication_expr`, `or_expr`, `and_expr`, `not_expr` | `parseExpr`, `parseImplication`, `parseOr`, `parseAnd`, `parseNot` | `TestParseExpressionPrecedence`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 25 | `predicate_expr`, `comparison_expr`, `comparison_op`, `membership_op` | `parsePredicate`, `matchComparisonOp` | `TestParseTestFile`, `TestParseExpressionPrecedence`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 25 | `value_expr` | `parseValueExpr`, `parseNamedValueExpr` | `TestParseNarrDeclarations`, `TestParseTestFile`, `TestParseExpressionPrecedence` | `TestParseErrorsContainLineAndColumn` | done |
| 26 | `existence_expr`, `temporal_predicate` | `parsePredicate`, `matchTemporalOrNarrativeOp` | `TestParseTestFile`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 26 | `aggregate_expr`, `aggregate_target` | `parseCountExpr`, `parseBinder` | `TestParseTestFile`, `TestParseExpressionPrecedence` | `TestParseErrorsContainLineAndColumn` | done |
| 27 | `state_expr`, `state_ref`, `chapter_boundary_anchor` | `parseStateExpr`, `parseRef` | `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 28 | `function_call`, `argument_list` | `parseNamedValueExpr`, `parseArgumentList` | `TestParseTestFile`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 29 | `narrative_predicate`, `service_predicate`, `promise_predicate`, `thread_predicate`, `arc_predicate`, `reveal_predicate`, `change_predicate` | `parsePredicate`, `matchTemporalOrNarrativeOp` | `TestParseTestFile`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | done |
| 29 | `narrative_subject`, `narr_structure_ref` | `parseValueExpr`, `parsePathExpr`, `parseRef` | `TestParseTestFile`, `TestParseExamplesProject` | `TestParseErrorsContainLineAndColumn` | partial |

Notes:

- M2 parses chapter and volume codes structurally as dotted refs. Canonicalization and legality checks belong to M3/M4.
- M2 parses time and anchor refs structurally as path refs. Target validation belongs to M3/M6.
- M2 parses narrative refs structurally. Type validation belongs to M3/M7.
