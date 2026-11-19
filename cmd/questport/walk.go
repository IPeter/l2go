package main

// inspect walks a statement or expression tree depth first. f is called
// for every node; returning false skips the node's children. Lambda bodies
// and anonymous class bodies are visited too.
func inspect(n any, f func(any) bool) {
	if n == nil {
		return
	}
	switch v := n.(type) {
	case stmt:
		if isNilStmt(v) || !f(v) {
			return
		}
		inspectStmt(v, f)
	case expr:
		if isNilExpr(v) || !f(v) {
			return
		}
		inspectExpr(v, f)
	case []stmt:
		for _, s := range v {
			inspect(s, f)
		}
	}
}

func isNilStmt(s stmt) bool {
	switch v := s.(type) {
	case *blockStmt:
		return v == nil
	}
	return s == nil
}

func isNilExpr(e expr) bool {
	switch v := e.(type) {
	case *arrayInitExpr:
		return v == nil
	}
	return e == nil
}

func inspectStmt(s stmt, f func(any) bool) {
	switch v := s.(type) {
	case *blockStmt:
		for _, c := range v.stmts {
			inspect(c, f)
		}
	case *localVarStmt:
		for _, d := range v.vars {
			if d.init != nil {
				inspect(d.init, f)
			}
		}
	case *exprStmt:
		inspect(v.x, f)
	case *ifStmt:
		inspect(v.cond, f)
		inspect(v.then, f)
		if v.els != nil {
			inspect(v.els, f)
		}
	case *switchStmt:
		inspect(v.tag, f)
		for _, c := range v.cases {
			for _, l := range c.labels {
				inspect(l, f)
			}
			for _, b := range c.body {
				inspect(b, f)
			}
		}
	case *forStmt:
		for _, i := range v.init {
			inspect(i, f)
		}
		if v.cond != nil {
			inspect(v.cond, f)
		}
		for _, u := range v.update {
			inspect(u, f)
		}
		inspect(v.body, f)
	case *forEachStmt:
		inspect(v.iter, f)
		inspect(v.body, f)
	case *whileStmt:
		inspect(v.cond, f)
		inspect(v.body, f)
	case *doWhileStmt:
		inspect(v.body, f)
		inspect(v.cond, f)
	case *returnStmt:
		if v.x != nil {
			inspect(v.x, f)
		}
	case *throwStmt:
		inspect(v.x, f)
	case *tryStmt:
		if v.body != nil {
			inspect(v.body, f)
		}
		for _, c := range v.catches {
			inspect(c.body, f)
		}
		if v.finally != nil {
			inspect(v.finally, f)
		}
	case *labeledStmt:
		inspect(v.body, f)
	case *syncStmt:
		inspect(v.lock, f)
		inspect(v.body, f)
	}
}

func inspectExpr(e expr, f func(any) bool) {
	switch v := e.(type) {
	case *parenExpr:
		inspect(v.x, f)
	case *fieldExpr:
		inspect(v.x, f)
	case *callExpr:
		if v.x != nil {
			inspect(v.x, f)
		}
		for _, a := range v.args {
			inspect(a, f)
		}
	case *indexExpr:
		inspect(v.x, f)
		inspect(v.index, f)
	case *newExpr:
		for _, a := range v.args {
			inspect(a, f)
		}
		for _, m := range v.body {
			if md, ok := m.(*methodDecl); ok && md.body != nil {
				inspect(md.body, f)
			}
		}
	case *newArrayExpr:
		for _, d := range v.dims {
			inspect(d, f)
		}
		if v.init != nil {
			inspect(v.init, f)
		}
	case *arrayInitExpr:
		for _, el := range v.elems {
			inspect(el, f)
		}
	case *unaryExpr:
		inspect(v.x, f)
	case *binaryExpr:
		inspect(v.x, f)
		inspect(v.y, f)
	case *assignExpr:
		inspect(v.x, f)
		inspect(v.y, f)
	case *condExpr:
		inspect(v.cond, f)
		inspect(v.then, f)
		inspect(v.els, f)
	case *castExpr:
		inspect(v.x, f)
	case *instanceOfExpr:
		inspect(v.x, f)
	case *lambdaExpr:
		if v.body != nil {
			inspect(v.body, f)
		}
		if v.block != nil {
			inspect(v.block, f)
		}
	case *methodRefExpr:
		inspect(v.x, f)
	}
}

// assignsName reports whether the tree assigns or increments the plain
// name.
func assignsName(n any, name string) bool {
	found := false
	inspect(n, func(x any) bool {
		if found {
			return false
		}
		switch v := x.(type) {
		case *assignExpr:
			if id, ok := unparen(v.x).(*identExpr); ok && id.name == name {
				found = true
			}
		case *unaryExpr:
			if v.op == "++" || v.op == "--" {
				if id, ok := unparen(v.x).(*identExpr); ok && id.name == name {
					found = true
				}
			}
		}
		return true
	})
	return found
}

// mentionsName reports whether the tree uses the plain name.
func mentionsName(n any, name string) bool {
	found := false
	inspect(n, func(x any) bool {
		if id, ok := x.(*identExpr); ok && id.name == name {
			found = true
		}
		return !found
	})
	return found
}

// hasSideEffects reports whether evaluating e may change state: calls
// (other than a few pure getters), assignments, increments, new objects.
func hasSideEffects(e expr) bool {
	found := false
	inspect(e, func(x any) bool {
		switch v := x.(type) {
		case *assignExpr:
			found = true
		case *unaryExpr:
			if v.op == "++" || v.op == "--" {
				found = true
			}
		case *callExpr:
			found = true
		case *newExpr:
			found = true
		}
		return !found
	})
	return found
}

// loopJumps finds the break and continue statements that leave the loop
// body s itself: those not inside a nested loop (or, for break, a nested
// switch).
func loopJumps(s stmt) (breaks, continues []stmt) {
	var walk func(n stmt, inSwitch bool)
	walk = func(n stmt, inSwitch bool) {
		switch v := n.(type) {
		case *breakStmt:
			if v.label == "" && !inSwitch {
				breaks = append(breaks, v)
			}
		case *continueStmt:
			if v.label == "" {
				continues = append(continues, v)
			}
		case *blockStmt:
			for _, c := range v.stmts {
				walk(c, inSwitch)
			}
		case *ifStmt:
			walk(v.then, inSwitch)
			if v.els != nil {
				walk(v.els, inSwitch)
			}
		case *switchStmt:
			for _, c := range v.cases {
				for _, b := range c.body {
					walk(b, true)
				}
			}
		case *labeledStmt:
			walk(v.body, inSwitch)
		case *tryStmt:
			walk(v.body, inSwitch)
			for _, c := range v.catches {
				walk(c.body, inSwitch)
			}
			if v.finally != nil {
				walk(v.finally, inSwitch)
			}
		case *syncStmt:
			walk(v.body, inSwitch)
		}
		// Nested loops own their own break and continue.
	}
	walk(s, false)
	return breaks, continues
}

// labeledJumps reports whether the tree has labeled break or continue.
func labeledJumps(n any) bool {
	found := false
	inspect(n, func(x any) bool {
		switch v := x.(type) {
		case *breakStmt:
			found = found || v.label != ""
		case *continueStmt:
			found = found || v.label != ""
		}
		return !found
	})
	return found
}

// endsAbruptly reports whether control cannot fall off the end of the
// statement list: it ends with break, continue, return or throw (also
// inside a final block or an if/else whose branches all end abruptly).
func endsAbruptly(stmts []stmt) bool {
	if len(stmts) == 0 {
		return false
	}
	return stmtEndsAbruptly(stmts[len(stmts)-1])
}

func stmtEndsAbruptly(s stmt) bool {
	switch v := s.(type) {
	case *breakStmt, *continueStmt, *returnStmt, *throwStmt:
		return true
	case *blockStmt:
		return endsAbruptly(v.stmts)
	case *ifStmt:
		return v.els != nil && stmtEndsAbruptly(v.then) && stmtEndsAbruptly(v.els)
	case *labeledStmt:
		return stmtEndsAbruptly(v.body)
	}
	return false
}
