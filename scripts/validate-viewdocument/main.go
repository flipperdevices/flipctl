package main

import (
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: validate-viewdocument <schema> <document>")
		os.Exit(2)
	}
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	f, err := os.Open(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	instance, err := jsonschema.UnmarshalJSON(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := schema.Validate(instance); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
