package catalogue

import (
	"context"
	"errors"
	"log"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
)

// Environment selects Acme's region for a named environment.
// Its runner accepts an optional logger through UseLogger before Run.
var Environment = job.Descriptor{
	Name:    "environment",
	Summary: "Select Acme's region for an environment.",
	MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		worker := new(environment)
		var inputs input.List
		inputs.StringVar(&worker.Name, "environment", "Environment name.")
		var outputs output.List
		outputs.StringVar(&worker.Region, "region", "Region for the environment.")
		return worker, &inputs, &outputs
	},
}

type environment struct {
	Name   string
	Region string
	logger *log.Logger
}

func (e *environment) UseLogger(logger *log.Logger) { e.logger = logger }

func (e *environment) Run(context.Context) error {
	switch e.Name {
	case "development":
		e.Region = "eu-west"
	case "production":
		e.Region = "eu-central"
	default:
		return errors.New("unknown environment")
	}
	if e.logger != nil {
		e.logger.Printf("environment: %s -> %s", e.Name, e.Region)
	}
	return nil
}

// Database derives an address from Acme's database naming rules.
// It does not provision a database or connect to one.
// Its runner accepts an optional logger through UseLogger before Run.
var Database = job.Descriptor{
	Name:    "database",
	Summary: "Derive a database address from Acme's naming rules.",
	MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		worker := new(database)
		var inputs input.List
		inputs.StringVar(&worker.Region, "region", "Database region.")
		var outputs output.List
		outputs.StringVar(&worker.Address, "address", "Database address.")
		return worker, &inputs, &outputs
	},
}

type database struct {
	Region  string
	Address string
	logger  *log.Logger
}

func (d *database) UseLogger(logger *log.Logger) { d.logger = logger }

func (d *database) Run(context.Context) error {
	if d.Region == "" {
		return errors.New("database region is empty")
	}
	d.Address = "database." + d.Region + ".example"
	if d.logger != nil {
		d.logger.Printf("database: %s", d.Address)
	}
	return nil
}

// API waits for shutdown after receiving its database address.
// Its runner accepts an optional logger through UseLogger before Run.
var API = serviceDescriptor("api")

// Reports waits for shutdown after receiving its database address.
// Its runner accepts an optional logger through UseLogger before Run.
var Reports = serviceDescriptor("reports")

func serviceDescriptor(name string) application.Descriptor {
	return application.Descriptor{
		Name:    name,
		Summary: "Keep " + name + " running until shutdown.",
		MakeFunc: func() (program.Runner, program.Inputs) {
			worker := &service{name: name}
			var inputs input.List
			inputs.StringVar(&worker.Database, "database", "Database address.")
			return worker, &inputs
		},
	}
}

type service struct {
	Database string
	name     string
	logger   *log.Logger
}

func (s *service) UseLogger(logger *log.Logger) { s.logger = logger }

func (s *service) Run(ctx context.Context) error {
	if s.Database == "" {
		return errors.New("database address is empty")
	}
	if s.logger != nil {
		s.logger.Printf("%s: started with %s", s.name, s.Database)
	}
	<-ctx.Done()
	if s.logger != nil {
		s.logger.Printf("%s: stopped", s.name)
	}
	return ctx.Err()
}
