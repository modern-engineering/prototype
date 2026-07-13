// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package completioncmd

import (
	"flag"
	"regexp"
	"sort"
	"strings"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

// A spec is the completion-relevant surface of one command, collected
// by walking the live dispatcher tree. The emitters render specs and
// never read base.Command themselves, so what completion knows about
// the CLI is exactly what one walk gathered.
type spec struct {
	name  string     // the dispatch word, base.Command.Name
	short string     // the one-line description, for shells that show one
	flags []flagSpec // a runnable command's flag surface, lexical order
	words []string   // literal first-argument alternatives, from the usage line
	files bool       // the command accepts file or directory arguments
	subs  []spec     // a command group's subcommands, help order
}

// A flagSpec is one flag of a runnable command.
type flagSpec struct {
	name    string // without the leading dash
	arg     string // the value's placeholder word; empty on a boolean flag
	usage   string // the help text, backquotes already stripped
	boolean bool   // a boolean flag completes with no following value
	repeats bool   // the usage advertises "(repeatable)"
}

// collect walks cmds the same way dispatch does: a command group
// contributes its subcommands, a runnable command its flag set and
// the positional surface of its usage line.
func collect(cmds []*base.Command) []spec {
	specs := make([]spec, 0, len(cmds))
	for _, cmd := range cmds {
		s := spec{name: cmd.Name(), short: cmd.Short}
		if len(cmd.Commands) > 0 {
			s.subs = collect(cmd.Commands)
			specs = append(specs, s)
			continue
		}
		cmd.Flag.VisitAll(func(f *flag.Flag) {
			arg, usage := flag.UnquoteUsage(f)
			b, ok := f.Value.(interface{ IsBoolFlag() bool })
			boolean := ok && b.IsBoolFlag()
			if boolean {
				arg = ""
			}
			s.flags = append(s.flags, flagSpec{
				name:    f.Name,
				arg:     arg,
				usage:   usage,
				boolean: boolean,
				// The flag package cannot say whether a flag
				// meaningfully repeats, so the usage text carries the
				// convention: a flag advertising "(repeatable)" is
				// offered again after it has been given once.
				repeats: strings.Contains(usage, "(repeatable)"),
			})
		})
		s.words, s.files = positionals(cmd.UsageLine, cmd.LongName())
		specs = append(specs, s)
	}
	return specs
}

// wordRE is the shape of a literal argument word in a usage-line
// enumeration like <vim|vscode>: bare lowercase words, nothing that
// could be a placeholder.
var wordRE = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// positionals parses a runnable command's usage-line trailer — the
// text after its command path — for the positional surface: a
// bracketed group of '|'-separated literal words (<vim|vscode>)
// enumerates the command's first argument, and any other non-flag
// placeholder ([dir], <image>, [path ...]) marks the command as
// taking file or directory arguments. Groups opening with a dash
// describe flags; VisitAll owns those, and this parser only skips
// them. The usage line is already walked material — Name derives
// from it — so reading the argument shape from the same line keeps
// the whole surface inside the dispatcher's own vocabulary.
func positionals(usageLine, longName string) (words []string, files bool) {
	trailer := strings.TrimPrefix(usageLine, "sdl "+longName)
	for {
		open := strings.IndexAny(trailer, "[<")
		if open < 0 {
			return words, files
		}
		closer := byte(']')
		if trailer[open] == '<' {
			closer = '>'
		}
		rest := trailer[open+1:]
		end := strings.IndexByte(rest, closer)
		if end < 0 {
			return words, files
		}
		chunk := rest[:end]
		trailer = rest[end+1:]
		switch {
		case strings.HasPrefix(chunk, "-"):
			// A flag group; the flag set owns it.
		case isEnumeration(chunk):
			words = append(words, strings.Split(chunk, "|")...)
		default:
			files = true
		}
	}
}

// isEnumeration reports whether chunk enumerates literal words.
func isEnumeration(chunk string) bool {
	parts := strings.Split(chunk, "|")
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		if !wordRE.MatchString(p) {
			return false
		}
	}
	return true
}

// The remaining helpers serve both emitters.

// names lists the dispatch words of specs, in tree order.
func names(specs []spec) []string {
	list := make([]string, len(specs))
	for i, s := range specs {
		list[i] = s.name
	}
	return list
}

// withHelp splices the dispatcher's help verb into a top-level name
// list. Help is dispatch-layer vocabulary — main handles it before
// consulting the command list — so the walker never sees it and the
// emitters add it here, at its alphabetical slot in the registration
// convention.
func withHelp(list []string) []string {
	i := sort.SearchStrings(list, "help")
	return append(list[:i:i], append([]string{"help"}, list[i:]...)...)
}

// fnameRE strips whatever a command name could carry that a shell
// function name cannot.
var fnameRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// fname renders a command path as the shell function name both
// emitters use for it.
func fname(path []string) string {
	name := "_sdl"
	if len(path) > 0 {
		name += "_" + strings.Join(path, "_")
	}
	return fnameRE.ReplaceAllString(name, "_")
}
