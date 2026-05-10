// Package lockblockingio provides an analysis.Analyzer that detects
// defer mu.Unlock()/RUnlock() followed by blocking operations in the
// same function body. This enforces CLAUDE.md Rule 1: "No Lock-During-Blocking-IO".
package lockblockingio

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/golangci/plugin-module-register/register"
)

func init() {
	register.Plugin("lockblockingio", newPlugin)
}

// plugin implements register.LinterPlugin.
type plugin struct{}

func newPlugin(_ any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}

func (p *plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}

// Analyzer detects defer mu.Unlock()/RUnlock() followed by blocking
// operations (channel send/recv, WaitGroup.Wait, blocking select)
// in the same function body.
var Analyzer = &analysis.Analyzer{
	Name: "lockblockingio",
	Doc: "detects defer mu.Unlock()/RUnlock() followed by blocking " +
		"operations (channel send/recv, WaitGroup.Wait, blocking select) " +
		"in the same function body",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		var body *ast.BlockStmt
		var funcName string

		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Body == nil {
				return
			}

			body = fn.Body
			funcName = fn.Name.Name
		case *ast.FuncLit:
			if fn.Body == nil {
				return
			}

			body = fn.Body
			funcName = "anonymous"
		default:
			return
		}

		checkFunctionBody(pass, body, funcName)
	})

	return nil, nil
}

// deferUnlock describes a defer statement that calls Unlock or RUnlock.
type deferUnlock struct {
	pos        token.Pos
	methodName string
}

// blockingOp describes a blocking operation found in a function body.
type blockingOp struct {
	pos         token.Pos
	description string
}

func checkFunctionBody(pass *analysis.Pass, body *ast.BlockStmt, funcName string) {
	unlocks := findDeferUnlocks(pass, body)
	if len(unlocks) == 0 {
		return
	}

	ops := findBlockingOps(pass, body)
	if len(ops) == 0 {
		return
	}

	// Report each defer-unlock paired with each blocking operation.
	for _, unlock := range unlocks {
		for _, op := range ops {
			pass.Reportf(
				unlock.pos,
				"defer %s followed by blocking operation %s at %s; "+
					"release lock before blocking (Rule 1)",
				unlock.methodName,
				op.description,
				pass.Fset.Position(op.pos),
			)
		}
	}
}

// findDeferUnlocks walks the function body for defer statements
// that call .Unlock() or .RUnlock() on a sync.Mutex or sync.RWMutex.
func findDeferUnlocks(pass *analysis.Pass, body *ast.BlockStmt) []deferUnlock {
	var results []deferUnlock

	ast.Inspect(body, func(n ast.Node) bool {
		ds, ok := n.(*ast.DeferStmt)
		if !ok {
			return true
		}

		call := ds.Call
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		methodName := sel.Sel.Name
		if methodName != "Unlock" && methodName != "RUnlock" {
			return true
		}

		if isMutexMethod(pass, sel) {
			results = append(results, deferUnlock{
				pos:        ds.Pos(),
				methodName: methodName,
			})
		}

		return true
	})

	return results
}

// isMutexMethod checks whether the selector expression refers to a method
// on sync.Mutex or sync.RWMutex.
func isMutexMethod(pass *analysis.Pass, sel *ast.SelectorExpr) bool {
	// Use type info to determine the receiver type.
	selObj := pass.TypesInfo.Selections[sel]
	if selObj != nil {
		recvType := selObj.Recv()

		return isSyncMutexType(recvType)
	}

	// Fallback: check if the object is a func declared on sync.Mutex/RWMutex.
	obj := pass.TypesInfo.ObjectOf(sel.Sel)
	if obj == nil {
		return false
	}

	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}

	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return false
	}

	recv := sig.Recv()
	if recv == nil {
		return false
	}

	return isSyncMutexType(recv.Type())
}

// isSyncMutexType checks if a type is *sync.Mutex or *sync.RWMutex
// (or their non-pointer variants).
func isSyncMutexType(t types.Type) bool {
	t = derefPointer(t)

	named, ok := t.(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj.Pkg() == nil {
		return false
	}

	if obj.Pkg().Path() != "sync" {
		return false
	}

	return obj.Name() == "Mutex" || obj.Name() == "RWMutex"
}

// findBlockingOps walks the function body for blocking operations:
// channel send, channel receive, WaitGroup.Wait, or blocking select.
// Channel sends/receives inside select case clauses are NOT reported
// separately -- only the select itself is reported (as blocking or not).
func findBlockingOps(pass *analysis.Pass, body *ast.BlockStmt) []blockingOp {
	var results []blockingOp

	ast.Inspect(body, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.SelectStmt:
			// Report the select as a whole if it is blocking (no default).
			// Do NOT descend into its case clauses -- channel ops there are
			// part of the select semantics and not independent blocking calls.
			if isBlockingSelect(stmt) {
				results = append(results, blockingOp{
					pos:         stmt.Pos(),
					description: "blocking select",
				})
			}

			return false // skip children

		case *ast.SendStmt:
			results = append(results, blockingOp{
				pos:         stmt.Pos(),
				description: "channel send",
			})

		case *ast.UnaryExpr:
			if stmt.Op == token.ARROW {
				results = append(results, blockingOp{
					pos:         stmt.Pos(),
					description: "channel receive",
				})
			}

		case *ast.ExprStmt:
			if isWaitGroupWait(pass, stmt) {
				results = append(results, blockingOp{
					pos:         stmt.Pos(),
					description: fmt.Sprintf("WaitGroup.Wait()"),
				})
			}
		}

		return true
	})

	return results
}

// isBlockingSelect returns true if the select statement has no default case.
func isBlockingSelect(sel *ast.SelectStmt) bool {
	for _, clause := range sel.Body.List {
		cc, ok := clause.(*ast.CommClause)
		if !ok {
			continue
		}

		if cc.Comm == nil {
			// default case.
			return false
		}
	}

	return true
}

// isWaitGroupWait checks if an expression statement is a call to
// sync.WaitGroup.Wait().
func isWaitGroupWait(pass *analysis.Pass, stmt *ast.ExprStmt) bool {
	call, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	if sel.Sel.Name != "Wait" {
		return false
	}

	// Check receiver type via selections.
	selObj := pass.TypesInfo.Selections[sel]
	if selObj == nil {
		return false
	}

	recvType := derefPointer(selObj.Recv())

	named, ok := recvType.(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj.Pkg() == nil {
		return false
	}

	return obj.Pkg().Path() == "sync" && obj.Name() == "WaitGroup"
}

// derefPointer strips pointer indirections from a type.
func derefPointer(t types.Type) types.Type {
	for {
		ptr, ok := t.(*types.Pointer)
		if !ok {
			return t
		}

		t = ptr.Elem()
	}
}
