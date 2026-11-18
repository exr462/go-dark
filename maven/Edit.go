package maven

import (
	"fmt"
	"os"
	"sort"
)

// Change sets the text of an edit location to Value.
type Change struct {
	Edit
	Value string
}

// Apply writes the changes, file by file. Each file is checked first: if
// any location no longer holds its expected text (the file changed since it
// was scanned), that file is left untouched and reported. Several changes
// to the same location (e.g. many dependencies sharing one property) are
// applied once; conflicting values for one location are an error.
func Apply(changes []Change) (changed []string, err error) {
	byFile := map[string]map[Span]Change{}
	for _, c := range changes {
		if c.Value == c.Current {
			continue
		}
		spans := byFile[c.File]
		if spans == nil {
			spans = map[Span]Change{}
			byFile[c.File] = spans
		}
		if prev, dup := spans[c.Span]; dup && prev.Value != c.Value {
			return nil, fmt.Errorf("%s: conflicting values %q and %q for the same version", c.File, prev.Value, c.Value)
		}
		spans[c.Span] = c
	}

	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)

	for _, file := range files {
		data, rerr := os.ReadFile(file)
		if rerr != nil {
			return changed, rerr
		}
		list := make([]Change, 0, len(byFile[file]))
		for _, c := range byFile[file] {
			if c.Span.End > len(data) || string(data[c.Span.Start:c.Span.End]) != c.Current {
				return changed, fmt.Errorf("%s changed since it was scanned; rescan and try again", file)
			}
			list = append(list, c)
		}
		// Apply back to front so earlier offsets stay valid.
		sort.Slice(list, func(i, j int) bool { return list[i].Span.Start > list[j].Span.Start })
		for _, c := range list {
			data = append(data[:c.Span.Start:c.Span.Start], append([]byte(c.Value), data[c.Span.End:]...)...)
		}
		info, serr := os.Stat(file)
		if serr != nil {
			return changed, serr
		}
		if werr := os.WriteFile(file, data, info.Mode().Perm()); werr != nil {
			return changed, werr
		}
		changed = append(changed, file)
	}
	return changed, nil
}
