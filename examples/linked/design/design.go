// Package design supplies structural intent without importing its catalogue.
package design

import "github.com/modern-engineering/prototype/ir"

const cataloguePath = "github.com/modern-engineering/prototype/examples/linked/catalogue"

// Model connects a database job to an application and includes a placement
// policy whose catalogue need not be understood by a program-aware consumer.
var Model = ir.Model{
	Nodes: []ir.Node{
		{Name: "database", Definition: ir.Symbol{Package: cataloguePath, Name: "Database"}},
		{Name: "backend", Definition: ir.Symbol{Package: cataloguePath, Name: "Backend"}},
		{Name: "placement", Definition: ir.Symbol{Package: cataloguePath, Name: "Europe"}},
	},
	Bindings: []ir.Binding{
		{Node: "database", Input: "region", Value: ir.Literal("eu-west")},
		{Node: "backend", Input: "database", Value: ir.Reference{Node: "database", Output: "address"}},
	},
}
