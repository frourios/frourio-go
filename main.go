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
		for rest := args[2:]; len(rest) > 0; {
			switch rest[0] {
			case "--openapi":
				if len(rest) < 2 {
					return fmt.Errorf("--openapi requires a path")
				}
				opts.OpenAPIPath = rest[1]
				rest = rest[2:]
			case "--watch":
				return fmt.Errorf("--watch is not implemented yet")
			default:
				return fmt.Errorf("unknown option: %s", rest[0])
			}
		}

		return generator.Generate(opts)
	case "openapi":
		if len(args) < 2 {
			return fmt.Errorf("usage: frourio-go openapi <api-dir> --output <path>")
		}

		opts := generator.Options{APIDir: args[1], OnlyOpenAPI: true}
		for rest := args[2:]; len(rest) > 0; {
			switch rest[0] {
			case "--output":
				if len(rest) < 2 {
					return fmt.Errorf("--output requires a path")
				}
				opts.OpenAPIPath = rest[1]
				rest = rest[2:]
			default:
				return fmt.Errorf("unknown option: %s", rest[0])
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
