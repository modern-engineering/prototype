package config_test

import (
	"fmt"
	"log"
	"strings"
	"time"

	"example.invalid/config"
)

// This example loads a configuration and reads values back, supplying a
// default for anything the file leaves unset.
func Example() {
	cfg, err := config.Load(strings.NewReader(`
# where the server listens
host = localhost
port = 8080
`))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cfg.String("host", "0.0.0.0"))
	fmt.Println(cfg.Int("port", 80))
	fmt.Println(cfg.Duration("timeout", 5*time.Second))
	// Output:
	// localhost
	// 8080
	// 5s
}
