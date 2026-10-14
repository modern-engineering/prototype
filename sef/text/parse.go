package text

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
)

type token struct {
	value string
	kind  byte
	pos   Position
}

// Parse reads a textual SEF file through EOF and leaves r open. filename is used
// only in diagnostics. I/O failures return no File and retain their original
// cause. Syntax failures return a partial File and an ErrorList; recovery resumes
// at the next line. Only the sef directive is required. Identity and revision
// default to empty strings when omitted.
func Parse(filename string, r io.Reader) (*File, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%s: read SEF: %w", filename, err)
	}
	file := new(File)
	var block string
	var blockStart Position
	seen := make(map[string]bool)
	offset := 0
	for number, source := range strings.Split(string(data), "\n") {
		position := Position{Filename: filename, Line: number + 1, Column: 1, Offset: offset}
		offset += len(source) + 1
		tokens, comment, err := lexLine(source, position)
		if err != nil {
			file.errors = append(file.errors, err)
			continue
		}
		if len(tokens) == 0 {
			file.lines = append(file.lines, line{comment: comment, inBlock: block != ""})
			continue
		}
		if tokens[0].kind == ')' {
			if block == "" || len(tokens) != 1 {
				file.diagnose(tokens[0].pos, "unexpected closing parenthesis")
				continue
			}
			file.lines = append(file.lines, line{kind: ')', comment: comment})
			block = ""
			continue
		}
		name := block
		arguments := tokens
		if block == "" {
			name = tokens[0].value
			if tokens[0].kind != 'w' {
				file.diagnose(tokens[0].pos, "expected directive")
				continue
			}
			arguments = tokens[1:]
			if len(arguments) == 1 && arguments[0].kind == '(' {
				if name != "require" && name != "node" && name != "bind" {
					file.diagnose(tokens[0].pos, "directive does not support a block: "+name)
				}
				block, blockStart = name, tokens[0].pos
				file.lines = append(file.lines, line{kind: '(', directive: name, comment: comment})
				continue
			}
		}
		values := make([]string, 0, len(arguments))
		valid := true
		for _, argument := range arguments {
			if argument.kind == '(' || argument.kind == ')' {
				file.diagnose(argument.pos, "unexpected parenthesis in statement")
				valid = false
				break
			}
			values = append(values, argument.value)
		}
		if !valid {
			continue
		}
		if file.apply(name, values, tokens[0].pos, seen) {
			file.lines = append(file.lines, line{kind: 's', directive: name, arguments: values, comment: comment, inBlock: block != ""})
		}
	}
	if block != "" {
		file.diagnose(blockStart, "unterminated "+block+" block")
	}
	if !seen["sef"] {
		file.diagnose(Position{Filename: filename, Line: 1, Column: 1}, "missing sef version directive")
	}
	if len(file.errors) != 0 {
		return file, file.errors
	}
	return file, nil
}

func (f *File) diagnose(position Position, message string) {
	f.errors = append(f.errors, &Error{Position: position, Message: message})
}

func (f *File) apply(name string, args []string, position Position, seen map[string]bool) bool {
	count := -1
	switch name {
	case "sef", "solution", "revision":
		count = 1
	case "require":
		count = 2
	case "node":
		count = 3
	case "bind":
		if len(args) >= 3 {
			switch args[2] {
			case "literal":
				count = 4
			case "reference":
				count = 5
			}
		}
		if count < 0 {
			f.diagnose(position, "bind expects an input followed by literal or reference")
			return false
		}
	default:
		f.diagnose(position, "unknown directive: "+name)
		return false
	}
	if len(args) != count {
		f.diagnose(position, fmt.Sprintf("%s expects %d arguments, found %d", name, count, len(args)))
		return false
	}
	if name == "sef" || name == "solution" || name == "revision" {
		if seen[name] {
			f.diagnose(position, "repeated "+name+" directive")
			return false
		}
		seen[name] = true
	}
	switch name {
	case "sef":
		f.artifact.Version = args[0]
		if args[0] != sef.Version {
			f.diagnose(position, "unsupported SEF version: "+strconv.Quote(args[0]))
			return false
		}
	case "solution":
		f.artifact.ID = args[0]
	case "revision":
		f.artifact.Revision = args[0]
	case "require":
		f.artifact.Definition.Modules = append(f.artifact.Definition.Modules, solution.Module{Path: args[0], Version: args[1]})
	case "node":
		f.nodes = append(f.nodes, position)
		f.artifact.Definition.Nodes = append(f.artifact.Definition.Nodes, solution.Node{
			Name: args[0], Definition: solution.Symbol{Package: args[1], Name: args[2]},
		})
	case "bind":
		var value solution.Value = solution.Literal(args[3])
		if args[2] == "reference" {
			value = solution.Reference{Node: args[3], Output: args[4]}
		}
		f.bindings = append(f.bindings, position)
		f.artifact.Definition.Bindings = append(f.artifact.Definition.Bindings, solution.Binding{Node: args[0], Input: args[1], Value: value})
	}
	return true
}

func lexLine(source string, start Position) ([]token, string, *Error) {
	var tokens []token
	for i := 0; i < len(source); {
		r, size := utf8.DecodeRuneInString(source[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		position := start
		position.Column += utf8.RuneCountInString(source[:i])
		position.Offset += i
		if strings.HasPrefix(source[i:], "//") {
			return tokens, strings.TrimRight(source[i:], " \t\r"), nil
		}
		if source[i] == '(' || source[i] == ')' {
			tokens = append(tokens, token{value: source[i : i+1], kind: source[i], pos: position})
			i++
			continue
		}
		begin := i
		if source[i] == '"' {
			i++
			for i < len(source) && source[i] != '"' {
				if source[i] == '\\' {
					i++
				}
				i++
			}
			if i >= len(source) {
				return nil, "", &Error{Position: position, Message: "unterminated quoted string"}
			}
			i++
			value, err := strconv.Unquote(source[begin:i])
			if err != nil {
				return nil, "", &Error{Position: position, Message: "invalid quoted string: " + err.Error()}
			}
			tokens = append(tokens, token{value: value, kind: 'q', pos: position})
			continue
		}
		for i < len(source) {
			r, size := utf8.DecodeRuneInString(source[i:])
			if unicode.IsSpace(r) || source[i] == '(' || source[i] == ')' || strings.HasPrefix(source[i:], "//") {
				break
			}
			i += size
		}
		tokens = append(tokens, token{value: source[begin:i], kind: 'w', pos: position})
	}
	return tokens, "", nil
}
