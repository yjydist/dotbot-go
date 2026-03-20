package output

import (
	"fmt"
	"io"
	"strings"
)

type Mode int

const (
	ModeDefault Mode = iota
	ModeVerbose
	ModeQuiet
)

type Status string

const (
	StatusInfo     Status = "info"
	StatusCreated  Status = "created"
	StatusLinked   Status = "linked"
	StatusSkipped  Status = "skipped"
	StatusReplaced Status = "replaced"
	StatusDeleted  Status = "deleted"
	StatusFailed   Status = "failed"
)

type Entry struct {
	Stage    string
	Target   string
	Source   string
	Decision string
	Status   Status
	Message  string
}

type Summary struct {
	Created  int
	Linked   int
	Skipped  int
	Replaced int
	Deleted  int
	Failed   int
}

func (s *Summary) Add(status Status) {
	switch status {
	case StatusCreated:
		s.Created++
	case StatusInfo:
	case StatusLinked:
		s.Linked++
	case StatusSkipped:
		s.Skipped++
	case StatusReplaced:
		s.Replaced++
	case StatusDeleted:
		s.Deleted++
	case StatusFailed:
		s.Failed++
	}
}

func WriteEntries(w io.Writer, mode Mode, dryRun bool, entries []Entry) {
	for _, entry := range entries {
		if mode == ModeQuiet && entry.Status != StatusFailed {
			continue
		}
		fmt.Fprintln(w, FormatEntry(dryRun, entry))
	}
}

func WriteSummary(w io.Writer, mode Mode, summary Summary) {
	if mode == ModeQuiet {
		return
	}
	fmt.Fprintf(w, "summary: created=%d linked=%d skipped=%d replaced=%d deleted=%d failed=%d\n", summary.Created, summary.Linked, summary.Skipped, summary.Replaced, summary.Deleted, summary.Failed)
}

func FormatEntry(dryRun bool, entry Entry) string {
	prefix := "[ok]"
	if dryRun {
		prefix = "[dry-run]"
	} else if entry.Status == StatusInfo {
		prefix = "[info]"
	} else if entry.Status == StatusSkipped {
		prefix = "[skip]"
	} else if entry.Status == StatusFailed {
		prefix = "[fail]"
	}
	object := entry.Target
	if entry.Source != "" {
		object = fmt.Sprintf("%s <- %s", entry.Target, entry.Source)
	}
	parts := []string{prefix, pad(entry.Stage, 7), pad(object, 40)}
	if entry.Decision != "" {
		parts = append(parts, entry.Decision)
	}
	if entry.Message != "" {
		parts = append(parts, fmt.Sprintf("(%s)", entry.Message))
	}
	return strings.Join(parts, " ")
}

func pad(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}
