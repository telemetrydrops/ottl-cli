// Copyright 2025 Dose de Telemetria GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// CLI command structure
var rootCmd = &cobra.Command{
	Use:   "ottl",
	Short: "A CLI tool for testing OTTL transformations",
	Long:  "ottl is a lean wrapper around the official OTTL library for testing transformations.",
}

var transformCmd = &cobra.Command{
	Use:   "transform",
	Short: "Apply OTTL transformation to OTLP data",
	Long: `Reads OTTL statement from stdin and applies it to the OTLP JSON data
in the specified input file. Outputs transformed OTLP JSON to stdout.`,
	Example: `  echo 'set(attributes["env"], "prod")' | ottl transform --input-file spans.json
  cat transform.ottl | ottl transform --input-file /path/to/spans.json`,
	RunE: runTransform,
}

var inputFile string

func init() {
	transformCmd.Flags().StringVarP(&inputFile, "input-file", "i", "", "Path to OTLP JSON input file (required)")
	transformCmd.MarkFlagRequired("input-file")
	rootCmd.AddCommand(transformCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runTransform executes the transform command
func runTransform(cmd *cobra.Command, args []string) error {
	// 1. Read OTTL statement from stdin
	ottlStatement, err := readStdin()
	if err != nil {
		return fmt.Errorf("failed to read OTTL statement from stdin: %w", err)
	}

	// 2. Read and parse input file using pdata JSON unmarshaler
	traces, err := readTracesFromFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// 3. Apply OTTL transformation
	err = applyOTTLTransformation(ottlStatement, traces)
	if err != nil {
		return fmt.Errorf("transformation failed: %w", err)
	}

	// 4. Output transformed data using pdata JSON marshaler
	if err := outputTransformedTraces(traces); err != nil {
		return fmt.Errorf("failed to output data: %w", err)
	}

	return nil
}

// readStdin reads OTTL statement from stdin
func readStdin() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	statement := strings.TrimSpace(strings.Join(lines, "\n"))
	if statement == "" {
		return "", fmt.Errorf("empty OTTL statement")
	}

	return statement, nil
}

// readTracesFromFile reads and parses OTLP JSON using pdata
func readTracesFromFile(filename string) (ptrace.Traces, error) {
	file, err := os.Open(filename)
	if err != nil {
		return ptrace.NewTraces(), fmt.Errorf("cannot open file %s: %w", filename, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return ptrace.NewTraces(), fmt.Errorf("cannot read file: %w", err)
	}

	// Use pdata JSON unmarshaler for official OTLP JSON format
	unmarshaler := &ptrace.JSONUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(data)
	if err != nil {
		return ptrace.NewTraces(), fmt.Errorf("invalid OTLP JSON format: %w", err)
	}

	return traces, nil
}

// applyOTTLTransformation applies the OTTL statement to the traces
func applyOTTLTransformation(statement string, traces ptrace.Traces) error {
	// Create OTTL parser for span context with standard functions
	parser, err := ottlspan.NewParser(ottlfuncs.StandardFuncs[ottlspan.TransformContext](), componenttest.NewNopTelemetrySettings())
	if err != nil {
		return fmt.Errorf("failed to create OTTL parser: %w", err)
	}

	// Parse the OTTL statement
	parsedStatement, err := parser.ParseStatement(statement)
	if err != nil {
		return fmt.Errorf("failed to parse OTTL statement '%s': %w", statement, err)
	}

	// Apply transformation to each span
	resourceSpans := traces.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		rs := resourceSpans.At(i)
		scopeSpans := rs.ScopeSpans()

		for j := 0; j < scopeSpans.Len(); j++ {
			ss := scopeSpans.At(j)
			spans := ss.Spans()

			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)

				// Create span context for OTTL
				spanCtx := ottlspan.NewTransformContext(span, ss.Scope(), rs.Resource(), ss, rs)

				// Execute the transformation
				_, _, err := parsedStatement.Execute(context.Background(), spanCtx)
				if err != nil {
					return fmt.Errorf("failed to execute transformation on span: %w", err)
				}
			}
		}
	}

	return nil
}

// outputTransformedTraces outputs traces as JSON using pdata marshaler
func outputTransformedTraces(traces ptrace.Traces) error {
	// Use pdata JSON marshaler for official OTLP JSON format
	marshaler := &ptrace.JSONMarshaler{}
	jsonData, err := marshaler.MarshalTraces(traces)
	if err != nil {
		return fmt.Errorf("failed to marshal traces to JSON: %w", err)
	}

	// Output to stdout
	fmt.Print(string(jsonData))
	return nil
}
