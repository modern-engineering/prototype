package output_test

import (
	"context"
	"fmt"

	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/program"
)

// The runner fills allocated output variables; their getters observe each update.
func Example() {
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		var outputs output.List
		status := outputs.String("status", "pending", "completion state")
		records := outputs.Int("records", uint64(0), "records processed")
		run := program.RunnerFunc(func(context.Context) error {
			*records = 3
			*status = "complete"
			return nil
		})
		return run, nil, &outputs
	})

	run, _, outputs := descriptor.Make()
	if err := run.Run(context.Background()); err != nil {
		fmt.Println(err)
		return
	}
	for field := range outputs.All() {
		fmt.Println(field.Name+":", field.Value.Get())
	}
	// Output:
	// status: complete
	// records: 3
}
