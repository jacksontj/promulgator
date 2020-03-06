package proxystorage

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/prometheus/promql"
)

// NewMultiVisitor takes a set of visitors and returns a MultiVisitor
func NewMultiVisitor(visitors []promql.Visitor) *MultiVisitor {
	return &MultiVisitor{
		visitors: visitors,
	}
}

// MultiVisitor runs a set of visitors on the same pass over the node tree
type MultiVisitor struct {
	l        sync.Mutex
	visitors []promql.Visitor
}

// Visit runs on each node in the tree
func (v *MultiVisitor) Visit(node promql.Node, path []promql.Node) (promql.Visitor, error) {
	var visitorErr error
	v.l.Lock()
	defer v.l.Unlock()
	for i, visitor := range v.visitors {
		if visitor == nil {
			continue
		}
		visitorRet, err := visitor.Visit(node, path)
		if err != nil {
			visitorErr = err
		}
		v.visitors[i] = visitorRet
	}

	return v, visitorErr

}

// OffsetFinder finds the offset (if any) within the tree
type OffsetFinder struct {
	l      sync.Mutex
	Found  bool
	Offset time.Duration
	Error  error
}

// Visit runs on each node in the tree
func (o *OffsetFinder) Visit(node promql.Node, _ []promql.Node) (promql.Visitor, error) {
	o.l.Lock()
	defer o.l.Unlock()
	switch n := node.(type) {
	case *promql.VectorSelector:
		if !o.Found {
			o.Offset = n.Offset
			o.Found = true
		} else {
			if n.Offset != o.Offset {
				o.Error = fmt.Errorf("mismatched offsets %v %v", n.Offset, o.Offset)
			}
		}

	case *promql.MatrixSelector:
		if !o.Found {
			o.Offset = n.Offset
			o.Found = true
		} else {
			if n.Offset != o.Offset {
				o.Error = fmt.Errorf("mismatched offsets %v %v", n.Offset, o.Offset)
			}
		}
	}
	if o.Error == nil {
		return o, nil
	}
	return nil, nil
}

// OffsetRemover removes any offset found in the node tree
// This is required when we send the queries below as we want to actually *remove* the offset.
type OffsetRemover struct{}

// Visit runs on each node in the tree
func (o *OffsetRemover) Visit(node promql.Node, _ []promql.Node) (promql.Visitor, error) {
	switch n := node.(type) {
	case *promql.VectorSelector:
		n.Offset = 0

	case *promql.MatrixSelector:
		n.Offset = 0
	}
	return o, nil
}

// BooleanFinder uses the given func to determine if something is in there or notret := &promql.VectorSelector{Offset: offset}
type BooleanFinder struct {
	Func  func(promql.Node) bool
	Found int
}

// Visit runs on each node in the tree
func (f *BooleanFinder) Visit(node promql.Node, _ []promql.Node) (promql.Visitor, error) {
	if f.Func(node) {
		f.Found++
		return f, nil
	}
	return f, nil
}

func CloneExpr(expr promql.Expr) (newExpr promql.Expr) {
	newExpr, _ = promql.ParseExpr(expr.String())
	return
}

// PreserveNameLabel wraps the input expression with a label replace in order to preserve the metadata through binary expressions
func PreserveNameLabel(expr promql.Expr, srcLabel string, dstLabel string) (relabelExpr promql.Expr) {
	relabelExpr, _ = promql.ParseExpr(fmt.Sprintf("label_replace(%s,`%s`,`$1`,`%s`,`(.*)`)", expr.String(), dstLabel, srcLabel))
	return relabelExpr
}
