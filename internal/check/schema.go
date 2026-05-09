package check

import (
	"fmt"

	"narr/internal/ast"
)

type fieldSpec struct {
	shape  fieldShape
	values map[string]bool
}

type fieldShape string

const (
	shapeAny          fieldShape = "any"
	shapeText         fieldShape = "text"
	shapeString       fieldShape = "string"
	shapeLanguage     fieldShape = "language"
	shapeInteger      fieldShape = "integer"
	shapeBool         fieldShape = "bool"
	shapeLength       fieldShape = "length"
	shapeRef          fieldShape = "ref"
	shapeIdentifier   fieldShape = "identifier"
	shapeList         fieldShape = "list"
	shapeSet          fieldShape = "set"
	shapeNarrLink     fieldShape = "narr_link"
	shapeLengthBlock  fieldShape = "length_block"
	shapeCondition    fieldShape = "condition_block"
	shapeEffect       fieldShape = "effect_block"
	shapeStartTargets fieldShape = "start_target_block"
	shapeHidden       fieldShape = "hidden_rule"
)

func (c *Checker) checkSchemas() {
	for _, file := range c.files {
		if file == nil || file.Mode != ast.ModeNarr {
			continue
		}
		for i := range file.Decls {
			c.checkDeclSchema(&file.Decls[i])
		}
	}
}

func (c *Checker) checkDeclSchema(decl *ast.Decl) {
	specs, ok := declFieldSpecs(decl.Kind)
	if !ok {
		return
	}
	for _, field := range decl.Fields {
		spec, ok := specs[field.Name]
		if !ok {
			c.addError("E0415", field.Span, fmt.Sprintf("%s does not support field %q", decl.Kind, field.Name))
			continue
		}
		c.checkFieldShape(decl.Kind, field, spec)
	}
}

func (c *Checker) checkFieldShape(kind ast.DeclKind, field ast.Field, spec fieldSpec) {
	if spec.shape == shapeLengthBlock && !fieldMatchesShape(field, spec.shape) {
		c.addError("E0418", field.Span, fmt.Sprintf("%s.%s contains an invalid length statement", kind, field.Name))
		return
	}
	if !fieldMatchesShape(field, spec.shape) {
		c.addError("E0416", field.Span, fmt.Sprintf("%s.%s must be %s", kind, field.Name, spec.shape))
		return
	}
	if len(spec.values) == 0 || field.Value == nil {
		return
	}
	if !spec.values[field.Value.Value] {
		c.addError("E0417", field.Value.Span, fmt.Sprintf("%s is not a valid value for %s.%s", field.Value.Value, kind, field.Name))
	}
}

func fieldMatchesShape(field ast.Field, shape fieldShape) bool {
	switch shape {
	case shapeAny:
		return field.Value != nil && len(field.Statements) == 0
	case shapeText:
		return field.Value != nil && (field.Value.Kind == ast.ExprString || field.Value.Kind == ast.ExprMultiline) && len(field.Statements) == 0
	case shapeString:
		return field.Value != nil && field.Value.Kind == ast.ExprString && len(field.Statements) == 0
	case shapeLanguage:
		return field.Value != nil && field.Value.Kind == ast.ExprLanguage && len(field.Statements) == 0
	case shapeInteger:
		return field.Value != nil && field.Value.Kind == ast.ExprInteger && len(field.Statements) == 0
	case shapeBool:
		return field.Value != nil && field.Value.Kind == ast.ExprBool && len(field.Statements) == 0
	case shapeLength:
		return field.Value != nil && field.Value.Kind == ast.ExprLength && len(field.Statements) == 0
	case shapeRef:
		return field.Value != nil && isReferenceExpr(field.Value) && len(field.Statements) == 0
	case shapeIdentifier:
		return field.Value != nil && field.Value.Kind == ast.ExprRef && len(field.Statements) == 0
	case shapeList:
		return field.Value != nil && field.Value.Kind == ast.ExprList && len(field.Statements) == 0
	case shapeSet:
		return field.Value != nil && field.Value.Kind == ast.ExprSet && len(field.Statements) == 0
	case shapeNarrLink:
		return field.Value != nil && isNarrLinkExpr(field.Value) && len(field.Statements) == 0
	case shapeLengthBlock:
		if field.Value != nil {
			return false
		}
		for _, stmt := range field.Statements {
			if stmt.Kind != ast.StmtLength || !validLengthStmt(stmt) {
				return false
			}
		}
		return true
	case shapeCondition:
		return field.Value == nil && allStatementsKind(field.Statements, ast.StmtCondition)
	case shapeEffect:
		return field.Value == nil && allEffectStatements(field.Statements)
	case shapeStartTargets:
		return field.Value == nil && allStatementsKind(field.Statements, ast.StmtStartTarget)
	case shapeHidden:
		return field.Value != nil && field.Value.Kind == ast.ExprBinary && field.Value.Op == "until" && len(field.Value.Children) == 2 && len(field.Statements) == 0
	default:
		return true
	}
}

func validLengthStmt(stmt ast.Stmt) bool {
	switch stmt.Name {
	case "volumes", "chapters_per_volume":
		return stmt.Value != nil && stmt.Value.Kind == ast.ExprInteger
	case "chapter":
		return stmt.Value != nil && stmt.Value.Kind == ast.ExprLength
	default:
		return false
	}
}

func allStatementsKind(statements []ast.Stmt, kind ast.StmtKind) bool {
	if len(statements) == 0 {
		return true
	}
	for _, stmt := range statements {
		if stmt.Kind != kind {
			return false
		}
	}
	return true
}

func allEffectStatements(statements []ast.Stmt) bool {
	if len(statements) == 0 {
		return true
	}
	for _, stmt := range statements {
		switch stmt.Kind {
		case ast.StmtAssignment, ast.StmtSetAdd, ast.StmtSetRemove, ast.StmtListAppend:
		default:
			return false
		}
	}
	return true
}

func isReferenceExpr(expr *ast.Expr) bool {
	return expr != nil && (expr.Kind == ast.ExprRef || expr.Kind == ast.ExprPath)
}

func isNarrLinkExpr(expr *ast.Expr) bool {
	if isReferenceExpr(expr) {
		return true
	}
	if expr == nil || (expr.Kind != ast.ExprList && expr.Kind != ast.ExprSet) {
		return false
	}
	for _, child := range expr.Children {
		if !isReferenceExpr(child) {
			return false
		}
	}
	return true
}

func declFieldSpecs(kind ast.DeclKind) (map[string]fieldSpec, bool) {
	switch kind {
	case ast.DeclNovel:
		return map[string]fieldSpec{
			"title":      {shape: shapeString},
			"language":   {shape: shapeLanguage},
			"summary":    {shape: shapeText},
			"length":     {shape: shapeLengthBlock},
			"prose_hint": {shape: shapeText},
		}, true
	case ast.DeclVolume:
		return map[string]fieldSpec{
			"title":           {shape: shapeString},
			"purpose":         {shape: shapeIdentifier, values: valueSet("setup", "escalation", "reversal", "descent", "revelation", "resolution", "interlude", "aftermath")},
			"summary":         {shape: shapeText},
			"target_chapters": {shape: shapeInteger},
			"target_length":   {shape: shapeLength},
		}, true
	case ast.DeclChapter:
		return map[string]fieldSpec{
			"title":         {shape: shapeString},
			"purpose":       {shape: shapeIdentifier, values: valueSet("entry", "encounter", "discovery", "conflict", "choice", "reversal", "aftermath", "reveal", "transition", "interlude", "climax", "quiet")},
			"start_pattern": {shape: shapeRef},
			"summary":       {shape: shapeText},
			"target_length": {shape: shapeLength},
			"pov":           {shape: shapeRef},
			"location":      {shape: shapeRef},
			"time_hint":     {shape: shapeText},
			"beats":         {shape: shapeList},
			"prose_hint":    {shape: shapeText},
		}, true
	case ast.DeclBeat:
		return map[string]fieldSpec{
			"summary":      {shape: shapeText},
			"precondition": {shape: shapeCondition},
			"effect":       {shape: shapeEffect},
			"pov":          {shape: shapeRef},
			"location":     {shape: shapeRef},
			"on_screen":    {shape: shapeBool},
			"observers":    {shape: shapeSet},
			"sets_up":      {shape: shapeNarrLink},
			"pays_off":     {shape: shapeNarrLink},
			"advances":     {shape: shapeNarrLink},
			"resolves":     {shape: shapeNarrLink},
			"reveals":      {shape: shapeNarrLink},
			"mentions":     {shape: shapeNarrLink},
			"render_hint":  {shape: shapeText},
		}, true
	case ast.DeclStartPattern:
		return map[string]fieldSpec{
			"at":       {shape: shapeRef},
			"requires": {shape: shapeCondition},
			"starts":   {shape: shapeStartTargets},
			"tags":     {shape: shapeSet},
			"note":     {shape: shapeText},
		}, true
	case ast.DeclPromise:
		return map[string]fieldSpec{
			"setup_at":          {shape: shapeRef},
			"start_pattern":     {shape: shapeRef},
			"setup_strength":    {shape: shapeIdentifier, values: valueSet("weak", "medium", "strong")},
			"payoff_by":         {shape: shapeRef},
			"payoff_at":         {shape: shapeRef},
			"payoff_kind":       {shape: shapeIdentifier, values: valueSet("answered", "reversed", "transformed_question", "emotional_payoff", "symbolic_payoff")},
			"question":          {shape: shapeText},
			"reader_visibility": {shape: shapeIdentifier, values: valueSet("hidden", "implied", "visible")},
			"tags":              {shape: shapeSet},
			"note":              {shape: shapeText},
		}, true
	case ast.DeclThread:
		return map[string]fieldSpec{
			"kind":                {shape: shapeIdentifier, values: valueSet("main_plot", "mystery", "romance", "political", "emotional", "thematic", "subplot")},
			"starts_at":           {shape: shapeRef},
			"start_pattern":       {shape: shapeRef},
			"expected_resolution": {shape: shapeRef},
			"resolved_at":         {shape: shapeRef},
			"priority":            {shape: shapeIdentifier, values: valueSet("main", "major", "minor", "background")},
			"tags":                {shape: shapeSet},
			"note":                {shape: shapeText},
		}, true
	case ast.DeclArc:
		return map[string]fieldSpec{
			"subject":             {shape: shapeRef},
			"starts_at":           {shape: shapeRef},
			"start_pattern":       {shape: shapeRef},
			"state_field":         {shape: shapeIdentifier},
			"initial":             {shape: shapeAny},
			"states":              {shape: shapeList},
			"expected_resolution": {shape: shapeRef},
			"tags":                {shape: shapeSet},
			"note":                {shape: shapeText},
		}, true
	case ast.DeclInvariant:
		return map[string]fieldSpec{
			"hidden":       {shape: shapeHidden},
			"always":       {shape: shapeCondition},
			"active_until": {shape: shapeRef},
			"tags":         {shape: shapeSet},
			"note":         {shape: shapeText},
		}, true
	default:
		return nil, false
	}
}

func valueSet(values ...string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		out[value] = true
	}
	return out
}
