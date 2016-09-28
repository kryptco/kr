// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package signature

import "sort"

// SortableMethods implements sort.Interface, ordering by method name.
type SortableMethods []Method

func (s SortableMethods) Len() int           { return len(s) }
func (s SortableMethods) Less(i, j int) bool { return s[i].Name < s[j].Name }
func (s SortableMethods) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// FindMethod returns the signature of the method with the given name and true
// iff the method exists, otherwise returns an empty signature and false.
func (s *Interface) FindMethod(name string) (Method, bool) {
	// Use linear rather than binary search, in case the methods aren't sorted.
	for _, method := range s.Methods {
		if method.Name == name {
			return method, true
		}
	}
	return Method{}, false
}

// FirstMethod returns the signature of the method with the given name and true
// iff the method exists, otherwise returns an empty signature and false.  If
// the method exists in more than one interface, we return the method from the
// the first interface with the given method name.
func FirstMethod(sig []Interface, name string) (Method, bool) {
	for _, s := range sig {
		if msig, ok := s.FindMethod(name); ok {
			return msig, true
		}
	}
	return Method{}, false
}

// CopyInterfaces returns a deep copy of x.
func CopyInterfaces(x []Interface) []Interface {
	if len(x) == 0 {
		return nil
	}
	ret := make([]Interface, len(x))
	for i, iface := range x {
		ret[i] = CopyInterface(iface)
	}
	return ret
}

// CopyInterface returns a deep copy of x.
func CopyInterface(x Interface) Interface {
	x.Embeds = CopyEmbeds(x.Embeds)
	x.Methods = CopyMethods(x.Methods)
	return x
}

// CopyEmbeds returns a deep copy of x.
func CopyEmbeds(x []Embed) []Embed {
	if len(x) == 0 {
		return nil
	}
	ret := make([]Embed, len(x))
	copy(ret, x)
	return ret
}

// CopyMethods returns a deep copy of x.
func CopyMethods(x []Method) []Method {
	if len(x) == 0 {
		return nil
	}
	ret := make([]Method, len(x))
	for i, method := range x {
		ret[i] = CopyMethod(method)
	}
	return ret
}

// CopyMethod returns a deep copy of x.
func CopyMethod(x Method) Method {
	x.InArgs = CopyArgs(x.InArgs)
	x.OutArgs = CopyArgs(x.OutArgs)
	x.InStream = copyArg(x.InStream)
	x.OutStream = copyArg(x.OutStream)
	return x
}

// CopyArgs returns a deep copy of x.
func CopyArgs(x []Arg) []Arg {
	if len(x) == 0 {
		return nil
	}
	ret := make([]Arg, len(x))
	copy(ret, x)
	return ret
}

// copyArg returns a deep copy of x.
func copyArg(x *Arg) *Arg {
	if x == nil {
		return nil
	}
	cp := *x
	return &cp
}

// MethodNames returns a sorted list of all method names from x.
func MethodNames(sig []Interface) []string {
	uniq := make(map[string]bool)
	for _, iface := range sig {
		for _, method := range iface.Methods {
			uniq[method.Name] = true
		}
	}
	var ret []string
	for name, _ := range uniq {
		ret = append(ret, name)
	}
	sort.Strings(ret)
	return ret
}

// CleanInterfaces returns a cleaned version of sig.  Duplicate interfaces are
// merged, duplicate embeds and methods are dropped, and all methods are sorted
// by name.
func CleanInterfaces(sig []Interface) []Interface {
	// First merge duplicate interfaces.
	ifaces := make(map[string]*Interface)
	for _, iface := range sig {
		key := interfaceKey(iface)
		if exist := ifaces[key]; exist != nil {
			mergeInterface(exist, iface)
		} else {
			n := new(Interface)
			*n = CopyInterface(iface)
			ifaces[key] = n
		}
	}
	// Now drop duplicate embeds and methods.
	for _, iface := range ifaces {
		iface.Embeds = dedupEmbeds(iface.Embeds)
		iface.Methods = dedupMethods(iface.Methods)
	}
	// Return interfaces, in the same relative order they were originally given.
	var ret []Interface
	for _, iface := range sig {
		key := interfaceKey(iface)
		if exist := ifaces[key]; exist != nil {
			ret = append(ret, *exist)
			delete(ifaces, key)
		}
	}
	return ret
}

func interfaceKey(x Interface) string {
	return x.PkgPath + "." + x.Name
}

func mergeInterface(dst *Interface, src Interface) {
	if dst.Doc == "" {
		dst.Doc = src.Doc
	}
	dst.Embeds = append(dst.Embeds, src.Embeds...)
	dst.Methods = append(dst.Methods, src.Methods...)
}

func dedupEmbeds(x []Embed) []Embed {
	seen := make(map[string]bool)
	cur, end := 0, len(x)
	for cur < end {
		if key := x[cur].PkgPath + "." + x[cur].Name; seen[key] {
			x[cur], x[end-1] = x[end-1], x[cur]
			end--
		} else {
			seen[key] = true
			cur++
		}
	}
	return x[:end]
}

func dedupMethods(x []Method) []Method {
	seen := make(map[string]bool)
	cur, end := 0, len(x)
	for cur < end {
		if key := x[cur].Name; seen[key] {
			x[cur], x[end-1] = x[end-1], x[cur]
			end--
		} else {
			seen[key] = true
			cur++
		}
	}
	ret := x[:end]
	sort.Sort(SortableMethods(ret))
	return ret
}
