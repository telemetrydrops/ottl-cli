package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Helper functions to read test data from external files
func readTestData(t *testing.T, filename string) []byte {
	data, err := os.ReadFile(filepath.Join("../../testdata", filename))
	require.NoError(t, err, "Failed to read test data file: %s", filename)
	return data
}

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
		t.Run(test.expected, func(t *testing.T) {
			assert.Equal(t, test.expected, test.ctx.String())
		})
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
		t.Run(test.flag, func(t *testing.T) {
			assert.Equal(t, test.expected, parseContextFlag(test.flag))
		})
	}
}

func TestDetectContextType(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		expectedType contextType
		shouldError  bool
	}{
		{
			name:         "valid traces",
			filename:     "traces.json",
			expectedType: contextTypeSpan,
			shouldError:  false,
		},
		{
			name:         "valid logs",
			filename:     "logs.json",
			expectedType: contextTypeLog,
			shouldError:  false,
		},
		{
			name:         "valid metrics",
			filename:     "metrics.json",
			expectedType: contextTypeMetric,
			shouldError:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := readTestData(t, test.filename)
			ctx, parsedData, err := detectContextType(data)

			if test.shouldError {
				assert.Error(t, err)
				assert.Equal(t, contextTypeUnknown, ctx)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectedType, ctx)
			assert.NotNil(t, parsedData)
		})
	}

	// Test error cases
	t.Run("empty JSON", func(t *testing.T) {
		ctx, _, err := detectContextType([]byte("{}"))
		assert.Error(t, err)
		assert.Equal(t, contextTypeUnknown, ctx)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		ctx, _, err := detectContextType([]byte("{invalid}"))
		assert.Error(t, err)
		assert.Equal(t, contextTypeUnknown, ctx)
	})
}

func TestParseDataWithContext(t *testing.T) {
	tracesData := readTestData(t, "traces.json")
	logsData := readTestData(t, "logs.json")
	metricsData := readTestData(t, "metrics.json")

	tests := []struct {
		name        string
		data        []byte
		ctx         contextType
		shouldError bool
	}{
		{
			name:        "parse traces as span context",
			data:        tracesData,
			ctx:         contextTypeSpan,
			shouldError: false,
		},
		{
			name:        "parse logs as log context",
			data:        logsData,
			ctx:         contextTypeLog,
			shouldError: false,
		},
		{
			name:        "parse metrics as metric context",
			data:        metricsData,
			ctx:         contextTypeMetric,
			shouldError: false,
		},
		{
			name:        "parse metrics as datapoint context",
			data:        metricsData,
			ctx:         contextTypeDatapoint,
			shouldError: false,
		},
		{
			name:        "invalid context type",
			data:        tracesData,
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
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, data)

			// Verify correct type is returned
			switch test.ctx {
			case contextTypeSpan:
				assert.IsType(t, ptrace.Traces{}, data)
			case contextTypeLog:
				assert.IsType(t, plog.Logs{}, data)
			case contextTypeMetric, contextTypeDatapoint:
				assert.IsType(t, pmetric.Metrics{}, data)
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
				defer func() { _ = w.Close() }()
				_, _ = w.Write([]byte(test.input))
			}()

			result, err := readStdin()

			// Restore stdin
			os.Stdin = oldStdin
			_ = r.Close()

			if test.shouldError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestReadInputFile(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "ottl_test_*.json")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	testData := "test file content"
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	_ = tmpFile.Close()

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
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expected, string(result))
		})
	}
}

func TestApplySpanTransformation(t *testing.T) {
	tracesData := readTestData(t, "traces.json")

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
			// Make a copy for each test to avoid side effects
			tracesCopy, err := (&ptrace.JSONUnmarshaler{}).UnmarshalTraces(tracesData)
			require.NoError(t, err)

			err = applySpanTransformation(test.statement, tracesCopy)

			if test.shouldError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify transformation was applied (for valid case)
			if test.statement == "set(attributes[\"env\"], \"test\")" {
				span := tracesCopy.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
				envAttr, exists := span.Attributes().Get("env")
				assert.True(t, exists, "Expected 'env' attribute to be set")
				assert.Equal(t, "test", envAttr.AsString())
			}
		})
	}
}

func TestApplyLogTransformation(t *testing.T) {
	logsData := readTestData(t, "logs.json")

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
			// Make a copy for each test to avoid side effects
			logsCopy, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(logsData)
			require.NoError(t, err)

			err = applyLogTransformation(test.statement, logsCopy)

			if test.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApplyMetricTransformation(t *testing.T) {
	metricsData := readTestData(t, "metrics.json")

	tests := []struct {
		name        string
		statement   string
		shouldError bool
	}{
		{
			name:        "valid metric transformation",
			statement:   "set(name, \"new_metric\")",
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
			// Make a copy for each test to avoid side effects
			metricsCopy, err := (&pmetric.JSONUnmarshaler{}).UnmarshalMetrics(metricsData)
			require.NoError(t, err)

			err = applyMetricTransformation(test.statement, metricsCopy)

			if test.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApplyDataPointTransformation(t *testing.T) {
	metricsData := readTestData(t, "metrics.json")

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
			// Make a copy for each test to avoid side effects
			metricsCopy, err := (&pmetric.JSONUnmarshaler{}).UnmarshalMetrics(metricsData)
			require.NoError(t, err)

			err = applyDataPointTransformation(test.statement, metricsCopy)

			if test.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOutputTransformedData(t *testing.T) {
	tracesData := readTestData(t, "traces.json")
	logsData := readTestData(t, "logs.json")
	metricsData := readTestData(t, "metrics.json")

	traces, err := (&ptrace.JSONUnmarshaler{}).UnmarshalTraces(tracesData)
	require.NoError(t, err)
	logs, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(logsData)
	require.NoError(t, err)
	metrics, err := (&pmetric.JSONUnmarshaler{}).UnmarshalMetrics(metricsData)
	require.NoError(t, err)

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
			_ = w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			output := buf.String()

			if test.shouldError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify we got some JSON output
			assert.NotEmpty(t, output)

			// Basic JSON validation - should start with { and end with }
			output = strings.TrimSpace(output)
			assert.True(t, strings.HasPrefix(output, "{"))
			assert.True(t, strings.HasSuffix(output, "}"))
		})
	}
}

// Integration test that exercises the full transform pipeline
func TestTransformIntegration(t *testing.T) {
	// Create temporary test files
	tracesData := readTestData(t, "traces.json")
	tracesFile, err := os.CreateTemp("", "traces_*.json")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tracesFile.Name()) }()

	_, err = tracesFile.Write(tracesData)
	require.NoError(t, err)
	_ = tracesFile.Close()

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
				defer func() { _ = w.Close() }()
				_, _ = w.Write([]byte(test.statement))
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
			_ = r.Close()
			_ = outW.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			_, _ = buf.ReadFrom(outR)

			if test.shouldError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			output := buf.String()
			assert.NotEmpty(t, output, "Expected some output from transformation")
		})
	}
}
