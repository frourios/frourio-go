package main

import (
	"fmt"
	"os"

	"github.com/frourios/frourio-go/internal/generator"
)

var osExit = os.Exit

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		osExit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: frourio-go generate <api-dir> [--openapi <path>] | frourio-go openapi <api-dir> --output <path>")
	}

	switch args[0] {
	case "generate":
		if len(args) < 2 {
			return fmt.Errorf("usage: frourio-go generate <api-dir> [--openapi <path>]")
		}

		opts := generator.Options{APIDir: args[1]}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--openapi":
				if i+1 >= len(args) {
					return fmt.Errorf("--openapi requires a path")
				}
				opts.OpenAPIPath = args[i+1]
				i++
			case "--watch":
				return fmt.Errorf("--watch is not implemented yet")
			default:
				return fmt.Errorf("unknown option: %s", args[i])
			}
		}

		return generator.Generate(opts)
	case "openapi":
		if len(args) < 2 {
			return fmt.Errorf("usage: frourio-go openapi <api-dir> --output <path>")
		}

		opts := generator.Options{APIDir: args[1], OnlyOpenAPI: true}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--output":
				if i+1 >= len(args) {
					return fmt.Errorf("--output requires a path")
				}
				opts.OpenAPIPath = args[i+1]
				i++
			default:
				return fmt.Errorf("unknown option: %s", args[i])
			}
		}

		if opts.OpenAPIPath == "" {
			return fmt.Errorf("openapi command requires --output")
		}
		return generator.Generate(opts)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}
