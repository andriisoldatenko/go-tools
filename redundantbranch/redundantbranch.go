// Copyright 2019 Axel Wagner
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package redundantbranch defines an Analyzer that checks for
// goto/breack/continue statements that don't affect control flow.
package redundantbranch

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `check for goto/break/continue statements that don't affect control flow

Examples are a break as the last statement in a case clause, a continue as the
last statement in a loop or a goto jumping to the next statement. We also take into account nested loops and statements.`

var Analyzer = &analysis.Analyzer{
	Name: "redundantbranch",
	Doc:  Doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	types := []ast.Node{
		new(ast.BranchStmt),
	}

	insp.WithStack(types, func(n ast.Node, push bool, stack []ast.Node) bool {
		branch := n.(*ast.BranchStmt)

		var ok bool
		switch branch.Tok {
		case token.BREAK:
			ok = checkBreak(pass, stack)
		case token.GOTO:
			ok = checkGoto(stack)
		case token.CONTINUE:
			ok = checkContinue(stack)
		case token.FALLTHROUGH:
			ok = true
		}
		if !ok {
			pass.Reportf(branch.Pos(), "%s does not affect control flow", strings.ToLower(branch.Tok.String()))
		}

		return false
	})

	return nil, nil
}

func checkGoto(stack []ast.Node) bool {
	branch := stack[len(stack)-1].(*ast.BranchStmt)

	if branch.Label == nil {
		panic("goto without label")
	}
	tgt := branch.Label.Obj.Decl.(*ast.LabeledStmt).Stmt
	next := nextStmt(branch, stack)
	return next != tgt
}

func checkBreak(pass *analysis.Pass, stack []ast.Node) bool {
	branch := stack[len(stack)-1].(*ast.BranchStmt)

	var tgt ast.Stmt
	if branch.Label != nil {
		tgt = branch.Label.Obj.Decl.(*ast.LabeledStmt).Stmt
	} else {
		for i := len(stack) - 2; i >= 0 && tgt == nil; i-- {
			switch st := stack[i].(type) {
			case *ast.ForStmt, *ast.RangeStmt, *ast.TypeSwitchStmt, *ast.SwitchStmt, *ast.SelectStmt:
				tgt = st.(ast.Stmt)
			}
		}
		if tgt == nil {
			panic("break outside of for/switch/select statement")
		}
	}

	tgt = nextStmt(tgt, stack)
	next := nextStmt(branch, stack)

	return next != tgt
}

func checkContinue(stack []ast.Node) bool {
	branch := stack[len(stack)-1].(*ast.BranchStmt)

	var tgt ast.Stmt
	if branch.Label != nil {
		tgt = branch.Label.Obj.Decl.(*ast.LabeledStmt).Stmt
	} else {
		for i := len(stack) - 2; i >= 0 && tgt == nil; i-- {
			switch st := stack[i].(type) {
			case *ast.ForStmt, *ast.RangeStmt:
				tgt = st.(ast.Stmt)
			}
		}
		if tgt == nil {
			panic("continue outside for statement")
		}
	}

	next := nextStmt(branch, stack)

	return next != tgt
}

// nextStmt returns the next statement executed after n (ignoring the control
// flow of n) or nil, if there is no such statement, because the function
// returns.
func nextStmt(n ast.Stmt, stack []ast.Node) (next ast.Stmt) {
	defer func() {
		if l, ok := next.(*ast.LabeledStmt); ok {
			next = l.Stmt
		}
	}()
	for len(stack) > 0 {
		st := stack[len(stack)-1]
		if n.Pos() > st.Pos() && n.End() <= st.End() {
			break
		}
		stack = stack[:len(stack)-1]
	}

	for i := len(stack) - 1; i >= 0; i-- {
		var list []ast.Stmt
		var parent ast.Stmt
		switch st := stack[i].(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			// last statement in function
			return nil
		case *ast.BlockStmt:
			list, parent = st.List, st
		case *ast.CaseClause:
			// CaseClause is surrounded by Block, surrounded by SwitchStmt.
			// Make the latter the parent.
			list, parent = st.Body, stack[i-2].(ast.Stmt)
		case *ast.CommClause:
			// CommClause is surrounded by Block, surrounded by SelectStmt. Make the latter the parent.
			list, parent = st.Body, stack[i-2].(ast.Stmt)
		case *ast.LabeledStmt:
			list, parent = []ast.Stmt{st.Stmt}, st
		case *ast.ForStmt:
			list, parent = []ast.Stmt{st.Body}, st
		case *ast.RangeStmt:
			list, parent = []ast.Stmt{st.Body}, st
		default:
			continue
		}

		for i, st := range list {
			if n.Pos() < st.Pos() || n.End() > st.End() {
				continue
			}
			if i < len(list)-1 {
				return list[i+1]
			}
			if _, ok := parent.(*ast.ForStmt); ok {
				return parent
			}
			if _, ok := parent.(*ast.RangeStmt); ok {
				return parent
			}
			return nextStmt(parent, stack)
		}
	}
	panic("statement not in tree")
}
