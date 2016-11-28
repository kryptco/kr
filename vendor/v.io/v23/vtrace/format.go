// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vtrace

import (
	"fmt"
	"io"
	"sort"
	"time"

	"v.io/v23/uniqueid"
)

const indentStep = "    "

type children []*Node

// children implements sort.Interface
func (c children) Len() int           { return len(c) }
func (c children) Less(i, j int) bool { return c[i].Span.Start.Before(c[j].Span.Start) }
func (c children) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

type annotations []Annotation

// annotations implements sort.Interface
func (a annotations) Len() int           { return len(a) }
func (a annotations) Less(i, j int) bool { return a[i].When.Before(a[j].When) }
func (a annotations) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type Node struct {
	Span     *SpanRecord
	Children []*Node
}

// TODO(mattr): It is useful in general to make a tree of spans
// for analysis as well as formatting.  This interface should
// be cleaned up and exported.
func BuildTree(trace *TraceRecord) *Node {
	var root *Node
	var earliestTime time.Time
	nodes := make(map[uniqueid.Id]*Node, len(trace.Spans))

	for i := range trace.Spans {
		span := &trace.Spans[i]
		if earliestTime.IsZero() || span.Start.Before(earliestTime) {
			earliestTime = span.Start
		}

		n := nodes[span.Id]
		if n == nil {
			n = &Node{}
			nodes[span.Id] = n
		}

		n.Span = span

		if span.Parent == trace.Id {
			root = n
		} else {
			p := nodes[span.Parent]
			if p == nil {
				p = &Node{}
				nodes[span.Parent] = p
			}
			p.Children = append(p.Children, n)
		}
	}

	// Sort the children of each node in start-time order, and the
	// annotation in time-order.
	for _, node := range nodes {
		sort.Sort(children(node.Children))
		if node.Span != nil {
			sort.Sort(annotations(node.Span.Annotations))
		}
	}

	// If we didn't find the root span of the trace
	// create a stand-in.
	if root == nil {
		root = &Node{
			Span: &SpanRecord{
				Name:  "Missing Root Span",
				Start: earliestTime,
			},
		}
	}

	// Find all nodes that have no span.  These represent missing data
	// in the tree.  We invent fake "missing" spans to represent
	// (perhaps several) layers of missing spans.  Then we add these as
	// children of the root.
	var missing []*Node
	for _, n := range nodes {
		if n.Span == nil {
			n.Span = &SpanRecord{
				Name: "Missing Data",
			}
			missing = append(missing, n)
		}
	}

	if len(missing) > 0 {
		root.Children = append(root.Children, missing...)
	}

	return root
}

func formatDelta(when, start time.Time) string {
	if start.IsZero() || when.IsZero() {
		return "??"
	}
	return when.Sub(start).String()
}

func formatNode(w io.Writer, n *Node, traceStart time.Time, indent string) {
	fmt.Fprintf(w, "%sSpan - %s [id: %x parent %x] (%s, %s: %s)\n",
		indent,
		n.Span.Name,
		n.Span.Id[12:],
		n.Span.Parent[12:],
		formatDelta(n.Span.Start, traceStart),
		formatDelta(n.Span.End, traceStart),
		formatDelta(n.Span.End, n.Span.Start))
	indent += indentStep
	for _, a := range n.Span.Annotations {
		fmt.Fprintf(w, "%s@%s %s\n", indent, formatDelta(a.When, traceStart), a.Message)
	}
	for _, c := range n.Children {
		formatNode(w, c, traceStart, indent)
	}
}

func formatTime(when time.Time, loc *time.Location) string {
	if when.IsZero() {
		return "??"
	}
	if loc != nil {
		when = when.In(loc)
	}
	return when.Format("2006-01-02 15:04:05.000000 MST")
}

// FormatTrace writes a text description of the given trace to the
// given writer.  Times will be formatted according to the given
// location, if loc is nil local times will be used.
func FormatTrace(w io.Writer, record *TraceRecord, loc *time.Location) {
	if root := BuildTree(record); root != nil {
		fmt.Fprintf(w, "Trace - %s (%s, %s)\n",
			record.Id,
			formatTime(root.Span.Start, loc),
			formatTime(root.Span.End, loc))
		for _, c := range root.Children {
			formatNode(w, c, root.Span.Start, indentStep)
		}
	}
}

// FormatTraces writes a text description of all the given traces to
// the given writer.  Times will be formatted according to the given
// location, if loc is nil local times will be used.
func FormatTraces(w io.Writer, records []TraceRecord, loc *time.Location) {
	if len(records) > 0 {
		fmt.Fprintf(w, "Vtrace traces:\n")
		for i := range records {
			FormatTrace(w, &records[i], loc)
		}
	}
}
