// Package maven reads the pom.xml files of the workspace: which artifacts
// each project produces, which workspace artifacts it references, and where
// exactly (byte offsets) each referenced version lives, so it can be edited
// without touching the rest of the file.
package maven

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// GA identifies an artifact (groupId:artifactId).
type GA struct {
	GroupID    string
	ArtifactID string
}

func (ga GA) String() string { return ga.GroupID + ":" + ga.ArtifactID }

// Span is a byte range in a file: the text inside an element.
type Span struct{ Start, End int }

// Valid reports whether the span points at real text.
func (s Span) Valid() bool { return s.End > s.Start }

// RefKind tells where a reference to another artifact appears.
type RefKind string

const (
	KindParent        RefKind = "parent"
	KindDependency    RefKind = "dependency"
	KindManaged       RefKind = "managed dependency"
	KindPlugin        RefKind = "plugin"
	KindManagedPlugin RefKind = "managed plugin"
)

// Coord is an artifact reference as written in a pom.
type Coord struct {
	GA
	Version     string // raw text, may be "${prop}"
	VersionSpan Span   // where Version is written (invalid if absent)
}

// Dependency is one dependency/plugin entry.
type Dependency struct {
	Coord
	Kind RefKind
}

// Property is one <properties> entry.
type Property struct {
	Value string
	Span  Span
}

// POM is the subset of a pom.xml go-dark needs.
type POM struct {
	Path         string
	Data         []byte
	Self         Coord // groupId/version are inherited from the parent when missing
	Parent       *Coord
	RelativePath string
	Properties   map[string]Property
	Dependencies []Dependency
}

// ParsePOM reads and parses a pom.xml.
func ParsePOM(path string) (*POM, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	p, err := parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	p.Path = path
	return p, nil
}

func parse(data []byte) (*POM, error) {
	p := &POM{Data: data, Properties: map[string]Property{}}
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false

	var stack []string
	var current *Dependency // dependency/plugin being read
	var text strings.Builder
	var textStart int

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			stack = append(stack, t.Name.Local)
			text.Reset()
			textStart = int(dec.InputOffset())
			if kind, ok := entryKind(stack); ok {
				current = &Dependency{Kind: kind}
			}
		case xml.CharData:
			text.Write(t)
		case xml.EndElement:
			end := int(dec.InputOffset()) - len("</"+t.Name.Local+">")
			value, span := trimmed(text.String(), textStart, end)
			// Only trust a span that holds exactly the value (not, say, a
			// comment inside the element); otherwise it isn't editable.
			if span.Valid() && (span.End > len(data) || string(data[span.Start:span.End]) != value) {
				span = Span{}
			}
			path := strings.Join(stack, "/")
			p.assign(path, value, span, &current, stack)
			if _, ok := entryKind(stack); ok && current != nil {
				if current.ArtifactID != "" {
					if current.GroupID == "" && (current.Kind == KindPlugin || current.Kind == KindManagedPlugin) {
						current.GroupID = "org.apache.maven.plugins"
					}
					p.Dependencies = append(p.Dependencies, *current)
				}
				current = nil
			}
			stack = stack[:len(stack)-1]
			text.Reset()
		}
	}

	if p.Parent != nil {
		if p.Self.GroupID == "" {
			p.Self.GroupID = p.Parent.GroupID
		}
		if p.Self.Version == "" {
			p.Self.Version = p.Parent.Version
		}
	}
	if p.Self.ArtifactID == "" {
		return nil, fmt.Errorf("no artifactId")
	}
	return p, nil
}

// entryKind reports whether the element path is a dependency/plugin entry.
func entryKind(stack []string) (RefKind, bool) {
	switch strings.Join(stack, "/") {
	case "project/dependencies/dependency":
		return KindDependency, true
	case "project/dependencyManagement/dependencies/dependency":
		return KindManaged, true
	case "project/build/plugins/plugin":
		return KindPlugin, true
	case "project/build/pluginManagement/plugins/plugin":
		return KindManagedPlugin, true
	}
	return "", false
}

func (p *POM) assign(path, value string, span Span, current **Dependency, stack []string) {
	switch path {
	case "project/groupId":
		p.Self.GroupID = value
	case "project/artifactId":
		p.Self.ArtifactID = value
	case "project/version":
		p.Self.Version, p.Self.VersionSpan = value, span
	case "project/parent/groupId":
		p.parent().GroupID = value
	case "project/parent/artifactId":
		p.parent().ArtifactID = value
	case "project/parent/version":
		p.parent().Version, p.parent().VersionSpan = value, span
	case "project/parent/relativePath":
		p.RelativePath = value
	}
	if len(stack) == 3 && stack[1] == "properties" {
		p.Properties[stack[2]] = Property{Value: value, Span: span}
	}
	if *current != nil && len(stack) >= 2 {
		if _, ok := entryKind(stack[:len(stack)-1]); ok {
			switch stack[len(stack)-1] {
			case "groupId":
				(*current).GroupID = value
			case "artifactId":
				(*current).ArtifactID = value
			case "version":
				(*current).Version, (*current).VersionSpan = value, span
			}
		}
	}
}

func (p *POM) parent() *Coord {
	if p.Parent == nil {
		p.Parent = &Coord{}
	}
	return p.Parent
}

// trimmed strips surrounding whitespace and adjusts the span to match.
func trimmed(s string, start, end int) (string, Span) {
	lead := len(s) - len(strings.TrimLeft(s, " \t\r\n"))
	trail := len(s) - len(strings.TrimRight(s, " \t\r\n"))
	v := strings.TrimSpace(s)
	if v == "" {
		return "", Span{}
	}
	return v, Span{Start: start + lead, End: end - trail}
}
