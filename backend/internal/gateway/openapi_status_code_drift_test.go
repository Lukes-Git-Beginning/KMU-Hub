package gateway_test

// TestOpenAPIStatusCodeDrift closes the third gap in the spec-vs-code guard set.
//
//   - TestOpenAPIRouteDrift  : registered path        subset of documented path
//   - TestOpenAPIMethodDrift : registered METHOD+path subset of documented METHOD+path
//   - this file              : status codes a handler really WRITES subset of the
//     status codes its operation documents
//
// Nothing checked the third direction. `swagger-cli validate` is structural
// and blind to a MISSING code, and the drift guards above stop at the
// operation key. The consequence was that every missing response code had to
// be found by a human reading handler and spec side by side - this run alone
// produced four such fix units, each covering only the routes that happened
// to be in its scope, out of 838 documented paths.
//
// How it works
//
//  1. Build the same chi.Router buildGatewayRouter already builds, walk it, and
//     recover each endpoint's Go function via runtime.FuncForPC on the bound
//     method value ("(*BizRoutes).HandleListInvoices-fm").
//  2. Parse the gateway package with go/parser and index every function and
//     method declaration under that same key.
//  3. Collect the status codes written by the known response writers
//     (response.JSON/Proto/ProtoList/ProtoListWrapped/Error, w.WriteHeader,
//     http.Error, http.Redirect), following calls into other functions of the
//     same package so helper-mediated codes count too: validateUUIDParam (400),
//     decodeAndValidate (400), validateDateParam (400),
//     respondServiceUnavailable (503) and ownerFilterForScope (401/403) are
//     reached by that recursion rather than by a hand-kept table.
//  4. Compare against the operation's `responses:` block in api/openapi.yaml.
//
// What it deliberately does NOT report
//
//   - respondGRPCError picks its code from the gRPC status of the service reply
//     (helpers.go:28, via grpcStatusToHTTP). That is not statically resolvable,
//     so the walk stops there instead of attributing 404/409/422/500 to the
//     handler. Reporting those would drown the real findings and get the test
//     switched off.
//   - a writer called with a non-literal status (a variable, a function result)
//     contributes nothing for the same reason.
//   - the reverse direction (documented but never written) is out of scope: with
//     respondGRPCError in the call graph one cannot prove a documented code is
//     unreachable.
//
// The accepted baseline below is today's remaining deviation set. Every entry
// is an open documentation gap, not an exemption on principle - the point of
// freezing it is that any NEW deviation fails immediately instead of waiting
// for the next person to read the spec by hand.

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// ---------------------------------------------------------------- handlers

// endpointFuncKey unwraps chi's middleware wrapper (r.With(...) stores the
// endpoint inside a *chi.ChainHandler) and returns the qualified name of the
// underlying gateway function in the form indexGatewayFuncs uses, e.g.
// "(*BizRoutes).HandleListInvoices". Returns "" for anything that is not a
// declared gateway function (closures, stdlib handlers).
func endpointFuncKey(h http.Handler) string {
	for range 8 {
		ch, ok := h.(*chi.ChainHandler)
		if !ok {
			break
		}
		h = ch.Endpoint
	}
	v := reflect.ValueOf(h)
	if v.Kind() != reflect.Func {
		return ""
	}
	fn := runtime.FuncForPC(v.Pointer())
	if fn == nil {
		return ""
	}
	const pkg = "github.com/kmuhub/kmuhub/internal/gateway."
	name := fn.Name() // e.g. <pkg>(*BizRoutes).HandleListInvoices-fm
	if !strings.HasPrefix(name, pkg) {
		return ""
	}
	name = strings.TrimSuffix(strings.TrimPrefix(name, pkg), "-fm")
	if strings.Contains(name, "func") {
		return "" // anonymous closure - no declaration to walk
	}
	return name
}

// ------------------------------------------------------------------- index

// gatewayFuncIndex holds the parsed gateway package: declarations by
// qualified key, a method-name to keys map used to resolve calls on a
// receiver variable without running the type checker, and the set of package
// identifiers imported anywhere in the package (so `time.Now()` is never
// mistaken for a local method call).
type gatewayFuncIndex struct {
	byKey       map[string]*ast.FuncDecl
	byMethod    map[string][]string
	importNames map[string]bool
}

// declKey renders a FuncDecl the same way endpointFuncKey renders a bound
// method value: "(*T).Method", "(T).Method" or plain "Function".
func declKey(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name
	}
	switch t := fd.Recv.List[0].Type.(type) {
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return "(*" + id.Name + ")." + fd.Name.Name
		}
	case *ast.Ident:
		return "(" + t.Name + ")." + fd.Name.Name
	}
	return fd.Name.Name
}

// indexGatewayFuncs parses every non-test .go file of the gateway package.
func indexGatewayFuncs(t *testing.T) *gatewayFuncIndex {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading the gateway package directory failed: %v", err)
	}

	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s failed: %v", name, err)
		}
		files = append(files, file)
	}

	idx := &gatewayFuncIndex{
		byKey:       make(map[string]*ast.FuncDecl),
		byMethod:    make(map[string][]string),
		importNames: make(map[string]bool),
	}
	for _, file := range files {
		for _, imp := range file.Imports {
			if imp.Name != nil {
				idx.importNames[imp.Name.Name] = true
				continue
			}
			path := strings.Trim(imp.Path.Value, `"`)
			idx.importNames[path[strings.LastIndex(path, "/")+1:]] = true
		}
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			key := declKey(fd)
			idx.byKey[key] = fd
			idx.byMethod[fd.Name.Name] = append(idx.byMethod[fd.Name.Name], key)
		}
	}
	if len(idx.byKey) == 0 {
		t.Fatal("indexed 0 gateway functions - the parser is broken or the package moved")
	}
	return idx
}

// ------------------------------------------------------------ code walking

// statusWriters maps a response-writing call to the argument position that
// carries the HTTP status. WriteHeader is handled separately because its
// receiver is a ResponseWriter under many different names.
var statusWriters = map[string]int{
	"response.JSON":             1,
	"response.Proto":            1,
	"response.ProtoList":        1,
	"response.ProtoListWrapped": 1,
	"response.Error":            1,
	"http.Error":                2,
	"http.Redirect":             3,
}

// dynamicStatusFuncs are gateway functions whose status code is decided at
// runtime. The walk stops at them instead of attributing their literal codes
// to the calling handler - see the file comment.
var dynamicStatusFuncs = map[string]bool{
	"respondGRPCError": true,
	"grpcStatusToHTTP": true,
}

// httpStatusValues resolves the http.StatusXxx constants a gateway handler
// can plausibly write. An unknown name resolves to nothing rather than being
// guessed.
var httpStatusValues = map[string]int{
	"StatusOK": 200, "StatusCreated": 201, "StatusAccepted": 202, "StatusNoContent": 204,
	"StatusPartialContent": 206,
	"StatusMovedPermanently": 301, "StatusFound": 302, "StatusSeeOther": 303,
	"StatusNotModified": 304, "StatusTemporaryRedirect": 307, "StatusPermanentRedirect": 308,
	"StatusBadRequest": 400, "StatusUnauthorized": 401, "StatusPaymentRequired": 402,
	"StatusForbidden": 403, "StatusNotFound": 404, "StatusMethodNotAllowed": 405,
	"StatusNotAcceptable": 406, "StatusRequestTimeout": 408, "StatusConflict": 409,
	"StatusGone": 410, "StatusPreconditionFailed": 412, "StatusRequestEntityTooLarge": 413,
	"StatusUnsupportedMediaType": 415, "StatusRequestedRangeNotSatisfiable": 416,
	"StatusUnprocessableEntity": 422, "StatusLocked": 423, "StatusFailedDependency": 424,
	"StatusTooManyRequests": 429,
	"StatusInternalServerError": 500, "StatusNotImplemented": 501, "StatusBadGateway": 502,
	"StatusServiceUnavailable": 503, "StatusGatewayTimeout": 504,
}

// callName renders a call target as ("pkg.Func" or "Func", bare selector).
// Generic instantiations (decodeAndValidate[T](...)) arrive as IndexExpr and
// are unwrapped. recvIdent is the receiver identifier of a selector call, ""
// when there is none.
func callName(fun ast.Expr) (qualified, bare, recvIdent string) {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name, f.Name, ""
	case *ast.IndexExpr:
		return callName(f.X)
	case *ast.IndexListExpr:
		return callName(f.X)
	case *ast.SelectorExpr:
		if x, ok := f.X.(*ast.Ident); ok {
			return x.Name + "." + f.Sel.Name, f.Sel.Name, x.Name
		}
		return "", f.Sel.Name, ""
	}
	return "", "", ""
}

// statusArg resolves a status argument to its numeric value. ok=false means
// the value is not statically known.
func statusArg(expr ast.Expr) (int, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.INT {
			if v, err := strconv.Atoi(e.Value); err == nil {
				return v, true
			}
		}
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok && x.Name == "http" {
			if v, ok := httpStatusValues[e.Sel.Name]; ok {
				return v, true
			}
		}
	}
	return 0, false
}

// writtenStatusCodes returns the status codes reachable from the function
// named by key, following calls into other functions of the same package.
func (idx *gatewayFuncIndex) writtenStatusCodes(key string, depth int, seen map[string]bool) map[int]bool {
	codes := make(map[int]bool)
	if depth > 6 || seen[key] {
		return codes
	}
	seen[key] = true

	fd, ok := idx.byKey[key]
	if !ok {
		return codes
	}

	ast.Inspect(fd.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		qualified, bare, recv := callName(call.Fun)

		if dynamicStatusFuncs[qualified] {
			return true // status decided at runtime - see file comment
		}

		pos, isWriter := statusWriters[qualified]
		if !isWriter && bare == "WriteHeader" {
			pos, isWriter = 0, true
		}
		if isWriter {
			if pos < len(call.Args) {
				if code, resolved := statusArg(call.Args[pos]); resolved {
					codes[code] = true
				}
			}
			return true
		}

		for _, cand := range idx.resolveCallee(qualified, bare, recv) {
			for c := range idx.writtenStatusCodes(cand, depth+1, seen) {
				codes[c] = true
			}
		}
		return true
	})

	return codes
}

// resolveCallee maps a call target to gateway declaration keys. A plain
// identifier is looked up directly; a call on a receiver variable is resolved
// by method name when that name is unique among the package's methods.
// Selectors whose receiver is an imported package name are never resolved.
func (idx *gatewayFuncIndex) resolveCallee(qualified, bare, recv string) []string {
	if recv == "" && qualified != "" && qualified == bare {
		if _, ok := idx.byKey[qualified]; ok {
			return []string{qualified}
		}
		return nil
	}
	if bare == "" || idx.importNames[recv] {
		return nil
	}
	var methods []string
	for _, k := range idx.byMethod[bare] {
		if strings.HasPrefix(k, "(") {
			methods = append(methods, k)
		}
	}
	if len(methods) == 1 {
		return methods
	}
	return nil // ambiguous, or a call into another package
}

// --------------------------------------------------------------- spec side

var (
	specPathKeyRE   = regexp.MustCompile(`^  (/\S+):\s*$`)
	specMethodKeyRE = regexp.MustCompile(`^    (get|post|put|patch|delete|head|options):\s*$`)
	// The spec mixes '200' and "200" quoting styles, so both are accepted.
	specStatusKeyRE = regexp.MustCompile(`^        ['"]?(\d{3})['"]?:`)
)

// documentedStatusCodes returns, per "METHOD /path", the set of response codes
// documented under that operation's responses: block. Same rigid
// two/four/six/eight-space indentation the sibling drift parsers rely on.
func documentedStatusCodes(t *testing.T, specPath string) map[string]map[int]bool {
	t.Helper()

	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", specPath, err)
	}

	out := make(map[string]map[int]bool)
	inPaths, inResponses := false, false
	currentPath, currentOp := "", ""

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if !inPaths {
			if strings.HasPrefix(line, "paths:") {
				inPaths = true
			}
			continue
		}
		if line != "" && line[0] != ' ' && line[0] != '\t' {
			break // a non-indented line ends the paths: map
		}
		if m := specPathKeyRE.FindStringSubmatch(line); m != nil {
			currentPath, currentOp, inResponses = m[1], "", false
			continue
		}
		if m := specMethodKeyRE.FindStringSubmatch(line); m != nil {
			currentOp = strings.ToUpper(m[1]) + " " + currentPath
			out[currentOp] = make(map[int]bool)
			inResponses = false
			continue
		}
		if currentOp == "" {
			continue
		}
		if line == "      responses:" {
			inResponses = true
			continue
		}
		// Any other key at operation level (exactly six spaces of indent) ends it.
		if inResponses && strings.HasPrefix(line, "      ") && len(line) > 6 && line[6] != ' ' {
			inResponses = false
		}
		if !inResponses {
			continue
		}
		if m := specStatusKeyRE.FindStringSubmatch(line); m != nil {
			code, _ := strconv.Atoi(m[1])
			out[currentOp][code] = true
		}
	}

	if len(out) == 0 {
		t.Fatalf("parsed 0 operations from %s - parser is broken or file moved", specPath)
	}
	return out
}

// ---------------------------------------------------------------- baseline

// systemicUndocumentedCodes are the two codes almost no operation documents,
// because almost every handler inherits them from the shared helpers rather
// than writing them itself: 400 from validateUUIDParam / decodeAndValidate /
// validateDateParam, and 503 from respondServiceUnavailable. At the time this
// guard was written they were missing from 347 and 1085 operations
// respectively - freezing that per operation would be an 1100-line literal
// nobody reads, and failing on it would leave the test red forever.
//
// So they are counted and logged instead of failed. Closing them is a
// spec-wide sweep of its own; once the count reaches zero, delete this map and
// the two codes become normal findings again.
var systemicUndocumentedCodes = map[int]string{
	400: "validateUUIDParam / decodeAndValidate / validateDateParam",
	503: "respondServiceUnavailable",
}

// statusDriftBaseline freezes what is left after the systemic codes: per
// "METHOD /path", the status codes the handler writes but the operation does
// not document. EVERY ENTRY IS AN OPEN DOCUMENTATION GAP, not an exemption on
// principle - removing one and watching this test stay green is how a spec fix
// is verified. Do not add to this list to silence a new finding; document the
// code in api/openapi.yaml instead.
//
// Reading the list: 401 on the HR routes comes from ownerFilterForScope
// (helpers.go:166, "missing user in token") or from getTenantID; 500 marks
// handlers that answer with a plain internal error instead of going through
// respondGRPCError.
var statusDriftBaseline = map[string][]int{
	"PUT /api/v1/customization/labels": {500},
}

// ------------------------------------------------------------------- dump
//
// OPENAPI_DRIFT_DUMP, when set, makes the test write every checked
// operation's full drift picture to that path as JSON, in addition to its
// normal pass/fail behavior. Unset (the default, including in CI), nothing
// below this comment runs and the test is exactly what it was before this
// dump existed. Consumed by
// .planning/backend-block/loop/hooks/openapi-status-fill.py, which inserts
// the missing responses into api/openapi.yaml so the gap does not have to be
// closed by hand, four to ten YAML lines at a time.

// driftDumpOperation is one entry of the dump. Unlike `findings` above,
// `Missing` is NOT filtered by statusDriftBaseline or systemicUndocumentedCodes
// - it lists every written code the spec does not document, because the fill
// hook needs the whole gap, not just the subset this test still fails on.
type driftDumpOperation struct {
	Method     string `json:"method"`
	Path       string `json:"path"`
	Written    []int  `json:"written"`
	Documented []int  `json:"documented"`
	Missing    []int  `json:"missing"`
	Mutating   bool   `json:"mutating"`
	Baselined  bool   `json:"baselined"`
}

type driftDump struct {
	Operations []driftDumpOperation `json:"operations"`
}

var mutatingHTTPMethods = map[string]bool{"POST": true, "PUT": true, "PATCH": true, "DELETE": true}

// newDriftDumpOperation renders one dump entry, written/documented/missing
// sorted so the JSON file is stable and diffable across runs.
func newDriftDumpOperation(method, path string, written, documented map[int]bool, baselined bool) driftDumpOperation {
	entry := driftDumpOperation{
		Method:     method,
		Path:       path,
		Mutating:   mutatingHTTPMethods[method],
		Baselined:  baselined,
		Written:    []int{},
		Documented: []int{},
		Missing:    []int{},
	}
	for c := range written {
		entry.Written = append(entry.Written, c)
		if !documented[c] {
			entry.Missing = append(entry.Missing, c)
		}
	}
	for c := range documented {
		entry.Documented = append(entry.Documented, c)
	}
	sort.Ints(entry.Written)
	sort.Ints(entry.Documented)
	sort.Ints(entry.Missing)
	return entry
}

// writeDriftDump marshals ops and writes them to path. Called only when
// OPENAPI_DRIFT_DUMP is set.
func writeDriftDump(t *testing.T, path string, ops []driftDumpOperation) {
	t.Helper()
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Path != ops[j].Path {
			return ops[i].Path < ops[j].Path
		}
		return ops[i].Method < ops[j].Method
	})
	data, err := json.MarshalIndent(driftDump{Operations: ops}, "", "  ")
	if err != nil {
		t.Fatalf("marshaling OPENAPI_DRIFT_DUMP failed: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing OPENAPI_DRIFT_DUMP to %s failed: %v", path, err)
	}
}

// TestOpenAPIStatusCodeDrift fails when a handler writes a status code its
// OpenAPI operation does not document and the pair is not in the frozen
// baseline above.
func TestOpenAPIStatusCodeDrift(t *testing.T) {
	r := buildGatewayRouter(t)
	idx := indexGatewayFuncs(t)
	documented := documentedStatusCodes(t, filepath.Join("..", "..", "api", "openapi.yaml"))

	baseline := make(map[string]map[int]bool, len(statusDriftBaseline))
	for op, codes := range statusDriftBaseline {
		set := make(map[int]bool, len(codes))
		for _, c := range codes {
			set[c] = true
		}
		baseline[op] = set
	}

	type finding struct {
		op    string
		codes []int
	}
	var findings []finding
	checked, unresolved := 0, 0
	systemic := make(map[int]int, len(systemicUndocumentedCodes))

	dumpPath := os.Getenv("OPENAPI_DRIFT_DUMP")
	var dumpOps []driftDumpOperation

	err := chi.Walk(r, func(method, route string, handler http.Handler, mws ...func(http.Handler) http.Handler) error {
		route = normalizeChiRoute(route)
		if !strings.HasPrefix(route, "/api/v1/") || !restVerbs[method] {
			return nil
		}
		op := method + " " + route
		docCodes, ok := documented[op]
		if !ok {
			return nil // undocumented operation - that is TestOpenAPIMethodDrift's job
		}
		key := endpointFuncKey(handler)
		if key == "" {
			unresolved++
			return nil
		}
		checked++

		writtenCodes := idx.writtenStatusCodes(key, 0, make(map[string]bool))

		var missing []int
		for code := range writtenCodes {
			if docCodes[code] || baseline[op][code] {
				continue
			}
			if _, isSystemic := systemicUndocumentedCodes[code]; isSystemic {
				systemic[code]++
				continue
			}
			missing = append(missing, code)
		}
		if len(missing) > 0 {
			sort.Ints(missing)
			findings = append(findings, finding{op: op, codes: missing})
		}

		if dumpPath != "" {
			_, baselined := statusDriftBaseline[op]
			dumpOps = append(dumpOps, newDriftDumpOperation(method, route, writtenCodes, docCodes, baselined))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk failed: %v", err)
	}

	if dumpPath != "" {
		writeDriftDump(t, dumpPath, dumpOps)
	}

	if len(findings) > 0 {
		sort.Slice(findings, func(i, j int) bool { return findings[i].op < findings[j].op })
		lines := make([]string, 0, len(findings))
		for _, f := range findings {
			lines = append(lines, fmt.Sprintf("%s writes %v - not documented", f.op, f.codes))
		}
		t.Errorf(
			"%d operation(s) write a status code api/openapi.yaml does not document "+
				"(add the response to the operation; do NOT extend statusDriftBaseline):\n  %s",
			len(findings), strings.Join(lines, "\n  "),
		)
	}

	for _, code := range sortedKeys(systemic) {
		t.Logf("systemic gap: %d undocumented on %d operation(s) — inherited from %s",
			code, systemic[code], systemicUndocumentedCodes[code])
	}
	t.Logf("checked %d documented operations against their handlers (%d handlers unresolved, %d baselined operations)",
		checked, unresolved, len(statusDriftBaseline))
}

// sortedKeys returns m's keys in ascending order, so the systemic-gap log is
// stable across runs.
func sortedKeys(m map[int]int) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

// TestOpenAPIStatusCodeDriftParserSanity checks both parsers this guard rests
// on, because either of them failing silently turns the guard green and
// useless. The spec side must find the codes of a long-stable operation; the
// code side must find both the status a known handler writes itself and the
// ones it only inherits from a helper, which is the part a naive AST scan of
// the handler body would miss.
func TestOpenAPIStatusCodeDriftParserSanity(t *testing.T) {
	documented := documentedStatusCodes(t, filepath.Join("..", "..", "api", "openapi.yaml"))

	login := documented["POST /api/v1/auth/login"]
	for _, want := range []int{200, 400, 401} {
		if !login[want] {
			t.Errorf("expected POST /api/v1/auth/login to document %d, parsed codes: %v", want, login)
		}
	}

	idx := indexGatewayFuncs(t)
	const handler = "(*ChatRoutes).HandleMarkChannelRead"
	written := idx.writtenStatusCodes(handler, 0, make(map[string]bool))
	for code, via := range map[int]string{
		204: "w.WriteHeader in the handler body",
		400: "validateUUIDParam / decodeAndValidate",
		503: "respondServiceUnavailable",
	} {
		if !written[code] {
			t.Errorf("expected %s to write %d (via %s), got %v", handler, code, via, written)
		}
	}
	// respondGRPCError must NOT contribute its mapped codes.
	if written[404] || written[500] {
		t.Errorf("%s picked up respondGRPCError's runtime codes, the dynamic cut-off is broken: %v", handler, written)
	}
}
