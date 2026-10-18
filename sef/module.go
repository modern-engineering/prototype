package sef

import "github.com/modern-engineering/prototype/solution"

type wireModuleContext struct {
	Path      string                  `json:"path"`
	Go        string                  `json:"go,omitempty"`
	Toolchain string                  `json:"toolchain,omitempty"`
	Require   []wireModule            `json:"require,omitempty"`
	Exclude   []wireModule            `json:"exclude,omitempty"`
	Replace   []wireModuleReplacement `json:"replace,omitempty"`
	Sum       []wireModuleSum         `json:"sum,omitempty"`
}
type wireModuleReplacement struct {
	Old wireModule `json:"old"`
	New wireModule `json:"new"`
}
type wireModuleSum struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

func marshalModule(m solution.ModuleContext) *wireModuleContext {
	if m.Path == "" && m.Go == "" && m.Toolchain == "" && len(m.Require)+len(m.Exclude)+len(m.Replace)+len(m.Sum) == 0 {
		return nil
	}
	result := &wireModuleContext{Path: m.Path, Go: m.Go, Toolchain: m.Toolchain}
	for _, r := range m.Require {
		result.Require = append(result.Require, wireModule(r))
	}
	for _, r := range m.Exclude {
		result.Exclude = append(result.Exclude, wireModule(r))
	}
	for _, r := range m.Replace {
		result.Replace = append(result.Replace, wireModuleReplacement{Old: wireModule(r.Old), New: wireModule(r.New)})
	}
	for _, sum := range m.Sum {
		result.Sum = append(result.Sum, wireModuleSum(sum))
	}
	return result
}

func unmarshalModule(m *wireModuleContext) solution.ModuleContext {
	if m == nil {
		return solution.ModuleContext{}
	}
	result := solution.ModuleContext{Path: m.Path, Go: m.Go, Toolchain: m.Toolchain}
	for _, r := range m.Require {
		result.Require = append(result.Require, solution.Module(r))
	}
	for _, r := range m.Exclude {
		result.Exclude = append(result.Exclude, solution.Module(r))
	}
	for _, r := range m.Replace {
		result.Replace = append(result.Replace, solution.ModuleReplacement{Old: solution.Module(r.Old), New: solution.Module(r.New)})
	}
	for _, sum := range m.Sum {
		result.Sum = append(result.Sum, solution.ModuleSum(sum))
	}
	return result
}
