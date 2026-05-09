package check

import (
	"fmt"
	"strings"

	"narr/internal/ast"
	"narr/internal/resolve"
	"narr/internal/source"
)

type semanticTypeKind string

const (
	semanticUnknown    semanticTypeKind = "unknown"
	semanticBool       semanticTypeKind = "bool"
	semanticInt        semanticTypeKind = "int"
	semanticString     semanticTypeKind = "string"
	semanticRef        semanticTypeKind = "ref"
	semanticAnchor     semanticTypeKind = "anchor"
	semanticList       semanticTypeKind = "list"
	semanticSet        semanticTypeKind = "set"
	semanticCollection semanticTypeKind = "collection"
	semanticObject     semanticTypeKind = "object"
)

type semanticType struct {
	Kind       semanticTypeKind
	Domain     string
	SymbolKind resolve.SymbolKind
}

type testSemanticChecker struct {
	checker *Checker
	env     resolve.FileEnv
	scopes  []map[string]semanticType
}

func (c *Checker) checkTestSemantics() {
	for _, file := range c.files {
		if file == nil || file.Mode != ast.ModeTest {
			continue
		}
		env := c.fileEnvs[file.Path]
		for i := range file.Tests {
			c.checkTestDeclSemantics(env, &file.Tests[i])
		}
	}
}

func (c *Checker) checkTestDeclSemantics(env resolve.FileEnv, test *ast.TestDecl) {
	checker := &testSemanticChecker{checker: c, env: env}
	checker.pushScope()
	defer checker.popScope()
	for _, attr := range test.Attrs {
		if attr.Name == "tags" && (attr.Value == nil || attr.Value.Kind != ast.ExprSet) {
			c.addError("E0902", attr.Span, "test tags must be a set")
		}
	}
	for i := range test.Statements {
		checker.checkStmt(&test.Statements[i])
	}
}

func (t *testSemanticChecker) checkStmt(stmt *ast.Stmt) {
	if stmt == nil {
		return
	}
	switch stmt.Kind {
	case ast.StmtAssert:
		exprType := t.infer(stmt.Expr)
		if !typeIs(exprType, semanticBool) {
			t.checker.addError("E0903", stmt.Span, "assert expression must be Bool")
		}
		t.infer(stmt.Message)
	case ast.StmtLet:
		t.infer(stmt.Value)
		t.defineLocal(stmt.Name, semanticType{Kind: semanticUnknown}, stmt.Span)
	case ast.StmtForall, ast.StmtExists:
		t.checkBinder(stmt.Binder)
		t.pushScope()
		if stmt.Binder != nil {
			t.defineLocal(stmt.Binder.Name, semanticType{Kind: semanticRef, Domain: stmt.Binder.Domain, SymbolKind: symbolKindForDomain(stmt.Binder.Domain)}, stmt.Binder.Span)
			whereType := t.infer(stmt.Binder.Where)
			if stmt.Binder.Where != nil && !typeIs(whereType, semanticBool) {
				t.checker.addError("E0903", stmt.Binder.Where.Span, "where expression must be Bool")
			}
		}
		for i := range stmt.Body {
			t.checkStmt(&stmt.Body[i])
		}
		t.popScope()
	}
}

func (t *testSemanticChecker) checkBinder(binder *ast.Binder) {
	if binder == nil {
		return
	}
	if binder.In == nil {
		return
	}
	collectionType := t.infer(binder.In)
	if collectionType.Kind == semanticUnknown {
		return
	}
	if collectionType.Kind != semanticCollection && collectionType.Kind != semanticList && collectionType.Kind != semanticSet {
		t.checker.addError("E0904", binder.In.Span, fmt.Sprintf("binder %s expects a collection", binder.Name))
		return
	}
	if collectionType.Domain != "" && collectionType.Domain != binder.Domain {
		t.checker.addError("E0904", binder.In.Span, fmt.Sprintf("binder %s expects %s values, got %s", binder.Name, binder.Domain, collectionType.Domain))
	}
}

func (t *testSemanticChecker) infer(expr *ast.Expr) semanticType {
	if expr == nil {
		return semanticType{Kind: semanticUnknown}
	}
	switch expr.Kind {
	case ast.ExprInvalid:
		return semanticType{Kind: semanticUnknown}
	case ast.ExprString, ast.ExprMultiline:
		return semanticType{Kind: semanticString}
	case ast.ExprInteger, ast.ExprLength:
		return semanticType{Kind: semanticInt}
	case ast.ExprBool:
		return semanticType{Kind: semanticBool}
	case ast.ExprLanguage:
		return semanticType{Kind: semanticString}
	case ast.ExprRef, ast.ExprPath:
		return t.inferRefPath(expr)
	case ast.ExprList:
		return t.inferCollectionLiteral(expr, semanticList)
	case ast.ExprSet:
		return t.inferCollectionLiteral(expr, semanticSet)
	case ast.ExprUnary:
		for _, child := range expr.Children {
			t.infer(child)
		}
		if expr.Op == "not" {
			return semanticType{Kind: semanticBool}
		}
	case ast.ExprBinary:
		return t.inferBinary(expr)
	case ast.ExprPostfix:
		for _, child := range expr.Children {
			t.infer(child)
		}
		if expr.Op == "exists" || expr.Op == "missing" {
			return semanticType{Kind: semanticBool}
		}
	case ast.ExprCall:
		return t.inferCall(expr)
	case ast.ExprMember:
		for _, child := range expr.Children {
			t.infer(child)
		}
		return semanticType{Kind: semanticUnknown}
	case ast.ExprCollection:
		return semanticType{Kind: semanticCollection, Domain: domainForCollectionName(expr.Value)}
	case ast.ExprCount:
		return t.inferCount(expr)
	case ast.ExprCollect:
		return t.inferCollect(expr)
	case ast.ExprState:
		for _, child := range expr.Children {
			t.infer(child)
		}
		return semanticType{Kind: semanticUnknown}
	case ast.ExprParen:
		if len(expr.Children) == 1 {
			return t.infer(expr.Children[0])
		}
	}
	return semanticType{Kind: semanticUnknown}
}

func (t *testSemanticChecker) inferRefPath(expr *ast.Expr) semanticType {
	if local, ok := t.lookupLocal(expr.Value); ok {
		return local
	}
	parts := strings.Split(expr.Value, ".")
	if len(parts) > 1 {
		if local, ok := t.lookupLocal(parts[0]); ok {
			if isAnchorSuffix(parts[len(parts)-1]) {
				return semanticType{Kind: semanticAnchor}
			}
			if local.Kind == semanticRef {
				return semanticType{Kind: semanticUnknown}
			}
		}
	}
	if expr.Value == "beginning" || expr.Value == "end_of_story" || isAnchorSuffix(parts[len(parts)-1]) {
		return semanticType{Kind: semanticAnchor}
	}
	symbol, _ := t.checker.resolved.ResolveName(t.env, expr.Value, expr.Span, false)
	if symbol == nil {
		return semanticType{Kind: semanticUnknown}
	}
	return semanticType{Kind: semanticRef, Domain: domainForSymbolKind(symbol.Kind), SymbolKind: symbol.Kind}
}

func (t *testSemanticChecker) inferCollectionLiteral(expr *ast.Expr, kind semanticTypeKind) semanticType {
	out := semanticType{Kind: kind}
	for _, child := range expr.Children {
		childType := t.infer(child)
		if childType.Domain == "" {
			continue
		}
		if out.Domain == "" {
			out.Domain = childType.Domain
			continue
		}
		if out.Domain != childType.Domain {
			out.Domain = ""
			return out
		}
	}
	return out
}

func (t *testSemanticChecker) inferBinary(expr *ast.Expr) semanticType {
	for _, child := range expr.Children {
		t.infer(child)
	}
	switch expr.Op {
	case "and", "or", "=>", "==", "!=", "<", "<=", ">", ">=", "in", "not in", "precedes", "at_or_before", "at_or_after", "between", "in_volume":
		return semanticType{Kind: semanticBool}
	case "serves", "mentions", "sets_up", "pays_off", "starts", "advances", "resolves", "reveals":
		t.checkNarrativePredicate(expr)
		return semanticType{Kind: semanticBool}
	case "changes":
		t.checkChangesPredicate(expr)
		return semanticType{Kind: semanticBool}
	default:
		return semanticType{Kind: semanticUnknown}
	}
}

func (t *testSemanticChecker) inferCall(expr *ast.Expr) semanticType {
	for _, arg := range expr.Args {
		t.infer(arg)
	}
	switch expr.Value {
	case "chapters_in":
		return semanticType{Kind: semanticCollection, Domain: "chapter"}
	case "beats":
		return semanticType{Kind: semanticCollection, Domain: "beat"}
	case "active_threads", "served_threads":
		return semanticType{Kind: semanticCollection, Domain: "thread"}
	case "active_promises", "served_promises":
		return semanticType{Kind: semanticCollection, Domain: "promise"}
	case "active_arcs", "served_arcs":
		return semanticType{Kind: semanticCollection, Domain: "arc"}
	case "reveals_in":
		return semanticType{Kind: semanticCollection, Domain: "fact"}
	case "mentions_in":
		return semanticType{Kind: semanticCollection}
	case "build":
		return semanticType{Kind: semanticObject}
	case "chapter_distance":
		return semanticType{Kind: semanticInt}
	case "chapters_between":
		return semanticType{Kind: semanticCollection, Domain: "chapter"}
	case "volume_of":
		return semanticType{Kind: semanticRef, Domain: "volume", SymbolKind: resolve.SymbolVolume}
	case "chapter_of", "previous", "next":
		return semanticType{Kind: semanticRef, Domain: "chapter", SymbolKind: resolve.SymbolChapter}
	case "canonical":
		return semanticType{Kind: semanticUnknown}
	default:
		return semanticType{Kind: semanticUnknown}
	}
}

func (t *testSemanticChecker) inferCount(expr *ast.Expr) semanticType {
	if expr.Binder != nil {
		t.checkBinder(expr.Binder)
		t.pushScope()
		t.defineLocal(expr.Binder.Name, semanticType{Kind: semanticRef, Domain: expr.Binder.Domain, SymbolKind: symbolKindForDomain(expr.Binder.Domain)}, expr.Binder.Span)
		whereType := t.infer(expr.Binder.Where)
		if expr.Binder.Where != nil && !typeIs(whereType, semanticBool) {
			t.checker.addError("E0903", expr.Binder.Where.Span, "where expression must be Bool")
		}
		t.popScope()
		return semanticType{Kind: semanticInt}
	}
	if len(expr.Children) != 1 {
		return semanticType{Kind: semanticInt}
	}
	target := t.infer(expr.Children[0])
	if target.Kind != semanticUnknown && target.Kind != semanticCollection && target.Kind != semanticList && target.Kind != semanticSet {
		t.checker.addError("E0904", expr.Children[0].Span, "count target must be a collection")
	}
	return semanticType{Kind: semanticInt}
}

func (t *testSemanticChecker) inferCollect(expr *ast.Expr) semanticType {
	if expr.Binder != nil {
		t.checkBinder(expr.Binder)
		t.pushScope()
		t.defineLocal(expr.Binder.Name, semanticType{Kind: semanticRef, Domain: expr.Binder.Domain, SymbolKind: symbolKindForDomain(expr.Binder.Domain)}, expr.Binder.Span)
		for _, child := range expr.Children {
			t.infer(child)
		}
		whereType := t.infer(expr.Binder.Where)
		if expr.Binder.Where != nil && !typeIs(whereType, semanticBool) {
			t.checker.addError("E0903", expr.Binder.Where.Span, "where expression must be Bool")
		}
		t.popScope()
	}
	return semanticType{Kind: semanticList}
}

func (t *testSemanticChecker) checkNarrativePredicate(expr *ast.Expr) {
	if len(expr.Children) < 2 {
		return
	}
	targetType := t.infer(expr.Children[1])
	if targetType.Kind == semanticUnknown {
		return
	}
	allowed := allowedNarrativeTargetKinds(expr.Op)
	if len(allowed) == 0 {
		return
	}
	for _, kind := range allowed {
		if targetType.SymbolKind == kind {
			return
		}
	}
	t.checker.addError("E0905", expr.Children[1].Span, fmt.Sprintf("%s target has type %s", expr.Op, targetType.SymbolKind))
}

func (t *testSemanticChecker) checkChangesPredicate(expr *ast.Expr) {
	if len(expr.Children) < 2 {
		return
	}
	fieldExpr := expr.Children[1]
	if fieldExpr == nil || !isReferenceExpr(fieldExpr) {
		t.checker.addError("E0905", expr.Span, "changes target must be a state field")
		return
	}
	if _, ok := t.checker.effectTarget(t.env, fieldExpr); !ok {
		t.checker.addError("E0905", fieldExpr.Span, "changes target must be a state field")
	}
}

func (t *testSemanticChecker) defineLocal(name string, value semanticType, span source.Span) {
	if name == "" {
		return
	}
	for index := len(t.scopes) - 1; index >= 0; index-- {
		if _, ok := t.scopes[index][name]; ok {
			t.checker.addError("E0901", span, fmt.Sprintf("duplicate test local %s", name))
			return
		}
	}
	if len(t.scopes) == 0 {
		t.pushScope()
	}
	t.scopes[len(t.scopes)-1][name] = value
}

func (t *testSemanticChecker) pushScope() {
	t.scopes = append(t.scopes, map[string]semanticType{})
}

func (t *testSemanticChecker) popScope() {
	if len(t.scopes) > 0 {
		t.scopes = t.scopes[:len(t.scopes)-1]
	}
}

func (t *testSemanticChecker) lookupLocal(path string) (semanticType, bool) {
	first, _, ok := strings.Cut(path, ".")
	if !ok {
		first = path
	}
	for index := len(t.scopes) - 1; index >= 0; index-- {
		if value, ok := t.scopes[index][first]; ok {
			return value, true
		}
	}
	return semanticType{}, false
}

func typeIs(value semanticType, kind semanticTypeKind) bool {
	return value.Kind == kind || value.Kind == semanticUnknown
}

func allowedNarrativeTargetKinds(op string) []resolve.SymbolKind {
	switch op {
	case "serves":
		return []resolve.SymbolKind{resolve.SymbolThread, resolve.SymbolPromise, resolve.SymbolArc, resolve.SymbolStartPattern}
	case "mentions":
		return nil
	case "sets_up", "pays_off":
		return []resolve.SymbolKind{resolve.SymbolPromise}
	case "starts":
		return []resolve.SymbolKind{resolve.SymbolThread, resolve.SymbolArc}
	case "advances":
		return []resolve.SymbolKind{resolve.SymbolThread, resolve.SymbolArc}
	case "resolves":
		return []resolve.SymbolKind{resolve.SymbolThread}
	case "reveals":
		return []resolve.SymbolKind{resolve.SymbolFact}
	default:
		return nil
	}
}

func symbolKindForDomain(domain string) resolve.SymbolKind {
	switch domain {
	case "novel":
		return resolve.SymbolNovel
	case "volume":
		return resolve.SymbolVolume
	case "chapter":
		return resolve.SymbolChapter
	case "beat":
		return resolve.SymbolBeat
	case "thread":
		return resolve.SymbolThread
	case "promise":
		return resolve.SymbolPromise
	case "arc":
		return resolve.SymbolArc
	case "character":
		return resolve.SymbolCharacter
	case "place":
		return resolve.SymbolPlace
	case "object":
		return resolve.SymbolObject
	case "fact":
		return resolve.SymbolFact
	default:
		return ""
	}
}

func domainForSymbolKind(kind resolve.SymbolKind) string {
	switch kind {
	case resolve.SymbolNovel:
		return "novel"
	case resolve.SymbolVolume:
		return "volume"
	case resolve.SymbolChapter:
		return "chapter"
	case resolve.SymbolBeat:
		return "beat"
	case resolve.SymbolThread:
		return "thread"
	case resolve.SymbolPromise:
		return "promise"
	case resolve.SymbolArc:
		return "arc"
	case resolve.SymbolCharacter:
		return "character"
	case resolve.SymbolPlace:
		return "place"
	case resolve.SymbolObject:
		return "object"
	case resolve.SymbolFact:
		return "fact"
	default:
		return ""
	}
}

func domainForCollectionName(name string) string {
	switch name {
	case "novels":
		return "novel"
	case "volumes":
		return "volume"
	case "chapters":
		return "chapter"
	case "beats":
		return "beat"
	case "threads":
		return "thread"
	case "promises":
		return "promise"
	case "arcs":
		return "arc"
	case "characters":
		return "character"
	case "places":
		return "place"
	case "objects":
		return "object"
	case "facts":
		return "fact"
	default:
		return ""
	}
}

func isAnchorSuffix(value string) bool {
	switch value {
	case "begin", "end", "before", "after":
		return true
	default:
		return false
	}
}
