package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Test data constants
const (
	validTracesJSON = `{
		"resourceSpans": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "test-service"}}]
			},
			"scopeSpans": [{
				"scope": {"name": "test-tracer", "version": "1.0.0"},
				"spans": [{
					"traceId": "0123456789abcdef0123456789abcdef",
					"spanId": "0123456789abcdef",
					"name": "test-span",
					"kind": 1,
					"startTimeUnixNano": "1609459200000000000",
					"endTimeUnixNano": "1609459201000000000",
					"attributes": [
						{"key": "http.method", "value": {"stringValue": "GET"}},
						{"key": "http.status_code", "value": {"intValue": "200"}}
					]
				}]
			}]
		}]
	}`

	validLogsJSON = `{
		"resourceLogs": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "test-service"}}]
			},
			"scopeLogs": [{
				"scope": {"name": "test-logger", "version": "1.0.0"},
				"logRecords": [{
					"timeUnixNano": "1609459200000000000",
					"severityNumber": 9,
					"severityText": "INFO",
					"body": {"stringValue": "This is a test log message"},
					"attributes": [
						{"key": "log.level", "value": {"stringValue": "INFO"}},
						{"key": "component", "value": {"stringValue": "auth"}}
					]
				}]
			}]
		}]
	}`

	validMetricsJSON = `{
		"resourceMetrics": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "test-service"}}]
			},
			"scopeMetrics": [{
				"scope": {"name": "test-meter", "version": "1.0.0"},
				"metrics": [{
					"name": "http_requests_total",
					"description": "Total HTTP requests",
					"unit": "1",
					"sum": {
						"dataPoints": [{
							"attributes": [{"key": "method", "value": {"stringValue": "GET"}}],
							"startTimeUnixNano": "1609459200000000000",
							"timeUnixNano": "1609459260000000000",
							"asInt": "150"
						}],
						"aggregationTemporality": 2,
						"isMonotonic": true
					}
				}]
			}]
		}]
	}`
)

func TestContextTypeString(t *testing.T) {
	tests := []struct {
		ctx      contextType
		expected string
	}{
		{contextTypeSpan, "span"},
		{contextTypeLog, "log"},
		{contextTypeMetric, "metric"},
		{contextTypeDatapoint, "datapoint"},
		{contextTypeUnknown, "unknown"},
	}

	for _, test := range tests {
		if got := test.ctx.String(); got != test.expected {
			t.Errorf("contextType(%d).String() = %s, want %s", test.ctx, got, test.expected)
		}
	}
}

func TestParseContextFlag(t *testing.T) {
	tests := []struct {
		flag     string
		expected contextType
	}{
		{"span", contextTypeSpan},
		{"SPAN", contextTypeSpan}, // case insensitive
		{"log", contextTypeLog},
		{"metric", contextTypeMetric},
		{"datapoint", contextTypeDatapoint},
		{"invalid", contextTypeUnknown},
		{"", contextTypeUnknown},
	}

	for _, test := range tests {
		if got := parseContextFlag(test.flag); got != test.expected {
			t.Errorf("parseContextFlag(%s) = %v, want %v", test.flag, got, test.expected)
		}
	}
}

func TestDetectContextType(t *testing.T) {
	tests := []struct {
		name         string
		data         []byte
		expectedType contextType
		shouldError  bool
	}{
		{
			name:         "valid traces",
			data:         []byte(validTracesJSON),
			expectedType: contextTypeSpan,
			shouldError:  false,
		},
		{
			name:         "valid logs",
			data:         []byte(validLogsJSON),
			expectedType: contextTypeLog,
			shouldError:  false,
		},
		{
			name:         "valid metrics",
			data:         []byte(validMetricsJSON),
			expectedType: contextTypeMetric,
			shouldError:  false,
		},
		{
			name:         "empty JSON",
			data:         []byte("{}"),
			expectedType: contextTypeUnknown,
			shouldError:  true,
		},
		{
			name:         "invalid JSON",
			data:         []byte("{invalid}"),
			expectedType: contextTypeUnknown,
			shouldError:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, data, err := detectContextType(test.data)

			if test.shouldError {
				if err == nil {
					t.Errorf("detectContextType() expected error but got none")
				}
				if ctx != contextTypeUnknown {
					t.Errorf("detectContextType() returned context %v, expected unknown on error", ctx)
				}
				return
			}

			if err != nil {
				t.Errorf("detectContextType() unexpected error: %v", err)
				return
			}

			if ctx != test.expectedType {
				t.Errorf("detectContextType() = %v, want %v", ctx, test.expectedType)
			}

			if data == nil {
				t.Errorf("detectContextType() returned nil data")
			}
		})
	}
}

func TestParseDataWithContext(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		ctx         contextType
		shouldError bool
	}{
		{
			name:        "parse traces as span context",
			data:        []byte(validTracesJSON),
			ctx:         contextTypeSpan,
			shouldError: false,
		},
		{
			name:        "parse logs as log context",
			data:        []byte(validLogsJSON),
			ctx:         contextTypeLog,
			shouldError: false,
		},
		{
			name:        "parse metrics as metric context",
			data:        []byte(validMetricsJSON),
			ctx:         contextTypeMetric,
			shouldError: false,
		},
		{
			name:        "parse metrics as datapoint context",
			data:        []byte(validMetricsJSON),
			ctx:         contextTypeDatapoint,
			shouldError: false,
		},
		{
			name:        "invalid context type",
			data:        []byte(validTracesJSON),
			ctx:         contextTypeUnknown,
			shouldError: true,
		},
		{
			name:        "invalid JSON for span context",
			data:        []byte("{invalid}"),
			ctx:         contextTypeSpan,
			shouldError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := parseDataWithContext(test.data, test.ctx)

			if test.shouldError {
				if err == nil {
					t.Errorf("parseDataWithContext() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("parseDataWithContext() unexpected error: %v", err)
				return
			}

			if data == nil {
				t.Errorf("parseDataWithContext() returned nil data")
				return
			}

			// Verify correct type is returned
			switch test.ctx {
			case contextTypeSpan:
				if _, ok := data.(ptrace.Traces); !ok {
					t.Errorf("parseDataWithContext() expected ptrace.Traces, got %T", data)
				}
			case contextTypeLog:
				if _, ok := data.(plog.Logs); !ok {
					t.Errorf("parseDataWithContext() expected plog.Logs, got %T", data)
				}
			case contextTypeMetric, contextTypeDatapoint:
				if _, ok := data.(pmetric.Metrics); !ok {
					t.Errorf("parseDataWithContext() expected pmetric.Metrics, got %T", data)
				}
			}
		})
	}
}

func TestReadStdin(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		shouldError bool
	}{
		{
			name:        "single line statement",
			input:       "set(attributes[\"env\"], \"prod\")",
			expected:    "set(attributes[\"env\"], \"prod\")",
			shouldError: false,
		},
		{
			name:        "multi-line statement",
			input:       "set(attributes[\"env\"], \"prod\")\nset(attributes[\"region\"], \"us-west\")",
			expected:    "set(attributes[\"env\"], \"prod\")\nset(attributes[\"region\"], \"us-west\")",
			shouldError: false,
		},
		{
			name:        "statement with leading/trailing whitespace",
			input:       "  set(attributes[\"env\"], \"prod\")  \n",
			expected:    "set(attributes[\"env\"], \"prod\")",
			shouldError: false,
		},
		{
			name:        "empty input",
			input:       "",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "whitespace only",
			input:       "   \n  \t  ",
			expected:    "",
			shouldError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Redirect stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Write test input
			go func() {
				defer w.Close()
				w.Write([]byte(test.input))
			}()

			result, err := readStdin()

			// Restore stdin
			os.Stdin = oldStdin
			r.Close()

			if test.shouldError {
				if err == nil {
					t.Errorf("readStdin() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("readStdin() unexpected error: %v", err)
				return
			}

			if result != test.expected {
				t.Errorf("readStdin() = %q, want %q", result, test.expected)
			}
		})
	}
}

func TestReadInputFile(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "ottl_test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testData := "test file content"
	if _, err := tmpFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	tests := []struct {
		name        string
		filename    string
		expected    string
		shouldError bool
	}{
		{
			name:        "valid file",
			filename:    tmpFile.Name(),
			expected:    testData,
			shouldError: false,
		},
		{
			name:        "non-existent file",
			filename:    "/non/existent/file.json",
			expected:    "",
			shouldError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := readInputFile(test.filename)

			if test.shouldError {
				if err == nil {
					t.Errorf("readInputFile() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("readInputFile() unexpected error: %v", err)
				return
			}

			if string(result) != test.expected {
				t.Errorf("readInputFile() = %q, want %q", string(result), test.expected)
			}
		})
	}
}

func TestApplySpanTransformation(t *testing.T) {
	traces, err := (&ptrace.JSONUnmarshaler{}).UnmarshalTraces([]byte(validTracesJSON))
	if err != nil {
		t.Fatalf("Failed to unmarshal traces: %v", err)
	}

	tests := []struct {
		name        string
		statement   string
		shouldError bool
	}{
		{
			name:        "valid span transformation",
			statement:   "set(attributes[\"env\"], \"test\")",
			shouldError: false,
		},
		{
			name:        "invalid OTTL syntax",
			statement:   "invalid_function()",
			shouldError: true,
		},
		{
			name:        "empty statement",
			statement:   "",
			shouldError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := applySpanTransformation(test.statement, traces)

			if test.shouldError {
				if err == nil {
					t.Errorf("applySpanTransformation() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("applySpanTransformation() unexpected error: %v", err)
				return
			}

			// Verify transformation was applied (for valid case)
			if test.statement == "set(attributes[\"env\"], \"test\")" {
				span := traces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
				envAttr, exists := span.Attributes().Get("env")
				if !exists {
					t.Errorf("Expected 'env' attribute to be set")
				} else if envAttr.AsString() != "test" {
					t.Errorf("Expected 'env' attribute to be 'test', got %s", envAttr.AsString())
				}
			}
		})
	}
}

func TestApplyLogTransformation(t *testing.T) {
	logs, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs([]byte(validLogsJSON))
	if err != nil {
		t.Fatalf("Failed to unmarshal logs: %v", err)
	}

	tests := []struct {
		name        string
		statement   string
		shouldError bool
	}{
		{
			name:        "valid log transformation",
			statement:   "set(attributes[\"env\"], \"test\")",
			shouldError: false,
		},
		{
			name:        "invalid OTTL syntax",
			statement:   "invalid_function()",
			shouldError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := applyLogTransformation(test.statement, logs)

			if test.shouldError {
				if err == nil {
					t.Errorf("applyLogTransformation() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("applyLogTransformation() unexpected error: %v", err)
				return
			}
		})
	}
}

func TestApplyMetricTransformation(t *testing.T) {
	metrics, err := (&pmetric.JSONUnmarshaler{}).UnmarshalMetrics([]byte(validMetricsJSON))
	if err != nil {
		t.Fatalf("Failed to unmarshal metrics: %v", err)
	}

	tests := []struct {
		name        string
		statement   string
		shouldError bool
	}{
		{
			name:        "valid metric transformation",
			statement:   "set(name, \"new_\" + name)",
			shouldError: false,
		},
		{
			name:        "invalid OTTL syntax",
			statement:   "invalid_function()",
			shouldError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := applyMetricTransformation(test.statement, metrics)

			if test.shouldError {
				if err == nil {
					t.Errorf("applyMetricTransformation() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("applyMetricTransformation() unexpected error: %v", err)
				return
			}
		})
	}
}

func TestApplyDataPointTransformation(t *testing.T) {
	metrics, err := (&pmetric.JSONUnmarshaler{}).UnmarshalMetrics([]byte(validMetricsJSON))
	if err != nil {
		t.Fatalf("Failed to unmarshal metrics: %v", err)
	}

	tests := []struct {
		name        string
		statement   string
		shouldError bool
	}{
		{
			name:        "valid datapoint transformation",
			statement:   "set(attributes[\"env\"], \"test\")",
			shouldError: false,
		},
		{
			name:        "invalid OTTL syntax",
			statement:   "invalid_function()",
			shouldError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := applyDataPointTransformation(test.statement, metrics)

			if test.shouldError {
				if err == nil {
					t.Errorf("applyDataPointTransformation() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("applyDataPointTransformation() unexpected error: %v", err)
				return
			}
		})
	}
}

func TestOutputTransformedData(t *testing.T) {
	traces, _ := (&ptrace.JSONUnmarshaler{}).UnmarshalTraces([]byte(validTracesJSON))
	logs, _ := (&plog.JSONUnmarshaler{}).UnmarshalLogs([]byte(validLogsJSON))
	metrics, _ := (&pmetric.JSONUnmarshaler{}).UnmarshalMetrics([]byte(validMetricsJSON))

	tests := []struct {
		name        string
		ctx         contextType
		data        interface{}
		shouldError bool
	}{
		{
			name:        "output traces",
			ctx:         contextTypeSpan,
			data:        traces,
			shouldError: false,
		},
		{
			name:        "output logs",
			ctx:         contextTypeLog,
			data:        logs,
			shouldError: false,
		},
		{
			name:        "output metrics",
			ctx:         contextTypeMetric,
			data:        metrics,
			shouldError: false,
		},
		{
			name:        "output datapoint",
			ctx:         contextTypeDatapoint,
			data:        metrics,
			shouldError: false,
		},
		{
			name:        "invalid context",
			ctx:         contextTypeUnknown,
			data:        traces,
			shouldError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := outputTransformedData(test.ctx, test.data)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if test.shouldError {
				if err == nil {
					t.Errorf("outputTransformedData() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("outputTransformedData() unexpected error: %v", err)
				return
			}

			// Verify we got some JSON output
			if len(output) == 0 {
				t.Errorf("outputTransformedData() produced no output")
			}

			// Basic JSON validation - should start with { and end with }
			output = strings.TrimSpace(output)
			if !strings.HasPrefix(output, "{") || !strings.HasSuffix(output, "}") {
				t.Errorf("outputTransformedData() produced invalid JSON: %s", output)
			}
		})
	}
}

// Integration test that exercises the full transform pipeline
func TestTransformIntegration(t *testing.T) {
	// Create temporary test files
	tracesFile, err := os.CreateTemp("", "traces_*.json")
	if err != nil {
		t.Fatalf("Failed to create traces temp file: %v", err)
	}
	defer os.Remove(tracesFile.Name())

	if _, err := tracesFile.WriteString(validTracesJSON); err != nil {
		t.Fatalf("Failed to write traces temp file: %v", err)
	}
	tracesFile.Close()

	tests := []struct {
		name        string
		inputFile   string
		statement   string
		shouldError bool
	}{
		{
			name:        "successful span transformation",
			inputFile:   tracesFile.Name(),
			statement:   "set(attributes[\"env\"], \"test\")",
			shouldError: false,
		},
		{
			name:        "non-existent input file",
			inputFile:   "/non/existent/file.json",
			statement:   "set(attributes[\"env\"], \"test\")",
			shouldError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set up stdin with OTTL statement
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			go func() {
				defer w.Close()
				w.Write([]byte(test.statement))
			}()

			// Capture stdout
			oldStdout := os.Stdout
			outR, outW, _ := os.Pipe()
			os.Stdout = outW

			// Set up the input file flag
			inputFile = test.inputFile
			contextFlag = ""

			// Call the main transformation function
			err := runTransform(nil, nil)

			// Restore stdin/stdout
			os.Stdin = oldStdin
			r.Close()
			outW.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(outR)

			if test.shouldError {
				if err == nil {
					t.Errorf("runTransform() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("runTransform() unexpected error: %v", err)
				return
			}

			output := buf.String()
			if len(output) == 0 {
				t.Errorf("runTransform() produced no output")
			}
		})
	}
}