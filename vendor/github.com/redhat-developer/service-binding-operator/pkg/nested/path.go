package nested

import (
	"strconv"
	"strings"
)

// field is the representation of a term in a field path.
//
// Assuming that 'status.dbCredentials' is a path for a value in a nested object, 'status' and
// 'dbCredentials' are represented as field values.
//
// field contains two members: 'Name' and 'Index'. 'Name' is a string like 'status' or
// 'dbCredentials', and 'Index' is the optional integer representation of the value, if it can be
// transformed to a valid positive integer.
type field struct {
	// Name is the field name.
	Name string
	// Index is the integer representation of the field name in the case it can be converted to an
	// integer value.
	Index *int
}

// newField creates a new field with the given name.
func newField(name string) field {
	f := field{Name: name}
	if i, err := strconv.Atoi(name); err == nil {
		f.Index = &i
	}
	return f
}

// path represents a field path.
type path []field

// Head returns the path head if one exists.
func (p path) head() (field, bool) {
	if len(p) > 0 {
		return p[0], true
	}
	return field{}, false
}

// tail returns the path tail if present.
//
// Returns 'b.c' in the path 'a.b.c'.
func (p path) tail() path {
	_, exists := p.head()
	if !exists {
		return path{}
	}
	return p[1:]
}

// hasTail asserts whether path has a tail.
func (p path) hasTail() bool {
	return len(p.tail()) > 0
}

// adjustedPath adjusts the current path depending on the head element.
//
// In the case the head of a path ('a' in the 'a.b.c' path) exists and is different than '*', returns
// itself otherwise returns the path tail ('b.c' in the example).
func (p path) adjustedPath() path {
	head, exists := p.head()
	if !exists {
		return path{}
	}
	if head.Name == "*" {
		return p.tail()
	}
	return p
}

// GetParts returns the path parts.
//
// For example, if the path contains the string 'a.b.c', returns a []string
// containing 'a', 'b' and 'c'.
func (p path) GetParts() []string {
	var parts []string
	clean := p.clean()
	for _, f := range clean {
		parts = append(parts, f.Name)
	}
	return parts
}

// lastField returns the last field from the receiver.
func (p path) lastField() (field, bool) {
	if len(p) > 0 {
		return p[len(p)-1], true
	}
	return field{}, false
}

// basePath returns the receiver's base path.
//
// For example, returns 'a.b' from the 'a.b.c' path.
func (p path) basePath() path {
	if len(p) > 1 {
		return p[:len(p)-1]
	}
	return path{}
}

// decompose returns the receiver's base path and the last field.
func (p path) decompose() (path, field) {
	f, _ := p.lastField()
	b := p.basePath()
	return b, f
}

// clean creates a new Path without '*' or integer values.
//
// For example, returns 'a.b.c' for 'a.b.*.c' or 'a.b.1.c'.
func (p path) clean() path {
	newPath := make(path, 0)
	for _, f := range p.adjustedPath() {
		if f.Index != nil {
			continue
		}
		if f.Name == "*" {
			continue
		}
		newPath = append(newPath, f)
	}
	return newPath
}

// NewPath creates a new path with the given string in the format 'a.b.c'.
func NewPath(s string) path {
	if s == "" {
		return path{}
	}
	parts := strings.Split(s, ".")
	return newPathWithParts(parts)
}

// NewPathWithParts constructs a Path from given parts.
func newPathWithParts(parts []string) path {
	p := make(path, len(parts))
	for i, part := range parts {
		p[i] = newField(part)
	}
	return p
}
