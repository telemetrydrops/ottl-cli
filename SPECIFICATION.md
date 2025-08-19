# Ultra-Lean Technical Specification: ottl-cli

## Overview

ottl-cli shall be an ultra-lean CLI wrapper around the official OpenTelemetry Transformation Language (OTTL) library, targeting exceptional simplicity while maintaining full functionality.

## Core Architecture Requirements

### Design Principles
- **Ultra-minimal codebase**: Single main.go file (target: 175 lines maximum)
- **Direct integration**: Must use official OTTL library types and functions directly
- **Simple CLI**: Clean cobra command structure  
- **Official package focus**: Direct usage of OpenTelemetry packages only

### Dependency Requirements
```go
// Maximum 4 dependencies - using official OTel packages only
require (
    github.com/spf13/cobra // CLI framework
    github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl // OTTL library
    go.opentelemetry.io/collector/pdata // OTLP data structures
    go.opentelemetry.io/collector/component // Testing utilities
)
```

## Required File Structure

```
ottl-cli/
├── cmd/ottl/main.go           # Complete application (target: 175 lines maximum)
├── go.mod                     # Maximum 4 direct dependencies
├── go.sum                     # Dependency checksums
├── Makefile                   # Build configuration
├── README.md                  # User documentation
├── PRD.md                     # Product requirements
├── SPECIFICATION.md           # Technical specification
└── local/payload-examples/    # Test data files
    ├── trace.json
    ├── logs.json
    ├── metrics.json
    └── events.json
```

## Implementation Requirements

### Main Application (`cmd/ottl/main.go`) - Target Implementation

The implementation shall use this structure with these required imports:

```go
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
    Use:   "ottl-cli",
    Short: "A CLI tool for testing OTTL transformations",
    Long:  "ottl-cli is a lean wrapper around the official OTTL library for testing transformations.",
}

var transformCmd = &cobra.Command{
    Use:   "transform",
    Short: "Apply OTTL transformation to OTLP data",
    Long: `Reads OTTL statement from stdin and applies it to the OTLP JSON data
in the specified input file. Outputs transformed OTLP JSON to stdout.`,
    Example: `  echo 'set(attributes["env"], "prod")' | ottl-cli transform --input-file spans.json
  cat transform.ottl | ottl-cli transform --input-file /path/to/spans.json`,
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

// applyOTTLTransformation applies the OTTL statement to the traces - ACTUAL CODE
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

```

## Error Handling

### Required Error Strategy
- **Basic error messages**: Clear, actionable error descriptions
- **Error context**: Include file names, line numbers when available
- **Remediation hints**: Simple suggestions for common issues
- **No custom error types**: Use standard Go error handling

### Common Error Scenarios
```go
// File not found
"Error: Cannot open file 'spans.json'\nRemediation: Ensure the file exists and is readable"

// Invalid JSON
"Error: Invalid JSON format at line 15\nRemediation: Validate JSON syntax using a JSON validator"

// Invalid OTTL syntax
"Error: Failed to parse OTTL statement 'set(invalid)'\nRemediation: Check OTTL syntax reference at https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/LANGUAGE.md"

// Transformation failure
"Error: Transformation failed on span 'my-span'\nRemediation: Verify the OTTL statement targets valid span fields"
```

## Testing Strategy

### Required Unit Tests
```go
func TestTransformCommand_BasicTransformation(t *testing.T) {
    // Test basic set attribute operation
    input := `{"resourceSpans":[{"resource":{"attributes":[]},"scopeSpans":[{"scope":{"name":"test"},"spans":[{"traceId":"1234","spanId":"5678","name":"test-span","attributes":[]}]}]}]}`
    ottl := `set(attributes["new"], "value")`
    
    // Create temp file with input
    tempFile := createTempFile(t, input)
    defer os.Remove(tempFile)
    
    // Execute command
    output, err := executeTransform(tempFile, ottl)
    require.NoError(t, err)
    
    // Verify output contains new attribute
    assert.Contains(t, output, `"new"`)
    assert.Contains(t, output, `"value"`)
}

func TestTransformCommand_InvalidOTTL(t *testing.T) {
    input := `{"resourceSpans":[]}`
    ottl := `invalid ottl syntax`
    
    tempFile := createTempFile(t, input)
    defer os.Remove(tempFile)
    
    _, err := executeTransform(tempFile, ottl)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "failed to parse OTTL statement")
}
```

## Build Configuration

### Makefile
```makefile
.PHONY: build test clean

BINARY_NAME := ottl-cli
BUILD_DIR := ./bin

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ottl

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)

example:
	echo 'set(attributes["example"], "true")' | $(BUILD_DIR)/$(BINARY_NAME) transform --input-file ./local/payload-examples/trace.json

fmt:
	go fmt ./...

lint:
	golangci-lint run
```

## Usage Examples

### Basic Transformation
```bash
# Set an attribute
echo 'set(attributes["env"], "prod")' | ottl-cli transform --input-file spans.json

# Delete an attribute  
echo 'delete_key(attributes, "internal.debug")' | ottl-cli transform --input-file spans.json

# Conditional transformation
echo 'set(attributes["processed"], true) where name == "http.request"' | ottl-cli transform --input-file spans.json
```

### Pipeline Usage
```bash
# Use with jq for specific output
cat transform.ottl | ottl-cli transform --input-file spans.json | jq '.resourceSpans[0].scopeSpans[0].spans[0].attributes'

# Save output to file
echo 'set(attributes["env"], "prod")' | ottl-cli transform --input-file input.json > output.json
```

## Performance Requirements

- **Startup time**: <50ms maximum
- **Small files** (<1MB): <100ms processing maximum
- **Memory usage**: <100MB for typical operations
- **Binary size**: <20MB maximum

## Key Requirements for Official Proto/Pdata Usage

1. **Zero custom JSON structs**: Must use official OpenTelemetry data types
2. **Perfect compatibility**: Must be 100% compatible with official OTLP JSON format
3. **Automatic updates**: Must stay current with OpenTelemetry specification changes
4. **Optimized marshaling**: Must use official high-performance JSON marshalers
5. **Reduced code size**: Target 175 lines maximum
6. **Better error handling**: Must use official parsers for validation and error messages
7. **Complete OTTL support**: Must include all standard functions through ottlfuncs.StandardFuncs
8. **Production ready**: Must include comprehensive error handling and performance

## Implementation Targets

This ultra-lean specification targets a 175-line single-file application that shall meet all performance requirements while providing complete OTTL functionality. The implementation shall demonstrate the power of using official OpenTelemetry packages for maximum compatibility and minimum maintenance overhead.

**Target Goals:**
- **Lines of code**: 175 maximum (vs typical 400-500 lines)
- **Dependencies**: 4 official packages maximum
- **Performance**: Must meet all specified requirements
- **Functionality**: Complete OTTL transformation support
- **Maintainability**: Single file, direct integration approach