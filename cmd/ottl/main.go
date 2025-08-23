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

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
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
	Long: `Reads OTTL statement from stdin and applies it to OTLP JSON data
in the specified input file. Supports traces, logs, and metrics with automatic
context detection. Outputs transformed OTLP JSON to stdout.`,
	Example: `  # Transform spans (auto-detected)
  echo 'set(attributes["env"], "prod")' | ottl transform --input-file spans.json

  # Transform logs (auto-detected)
  echo 'set(log.severity_text, "ERROR")' | ottl transform --input-file logs.json

  # Transform metrics (auto-detected)
  echo 'set(metric.name, "custom_" + metric.name)' | ottl transform --input-file metrics.json

  # Force specific context
  echo 'set(datapoint.value_double, 0)' | ottl transform --input-file metrics.json --context datapoint

  # From file
  cat transform.ottl | ottl transform --input-file /path/to/data.json`,
	RunE: runTransform,
}

// contextType represents the different OTTL contexts
type contextType int

const (
	contextTypeUnknown contextType = iota
	contextTypeSpan
	contextTypeLog
	contextTypeMetric
	contextTypeDatapoint
)

func (c contextType) String() string {
	switch c {
	case contextTypeSpan:
		return "span"
	case contextTypeLog:
		return "log"
	case contextTypeMetric:
		return "metric"
	case contextTypeDatapoint:
		return "datapoint"
	default:
		return "unknown"
	}
}

var inputFile string
var contextFlag string

func init() {
	transformCmd.Flags().StringVarP(&inputFile, "input-file", "i", "", "Path to OTLP JSON input file (required)")
	transformCmd.Flags().StringVar(&contextFlag, "context", "", "Force specific OTTL context (span, log, metric, datapoint)")
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

	// 2. Read input file and detect context type
	data, err := readInputFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	ctx, parsedData, err := detectContextType(data)
	if err != nil {
		return fmt.Errorf("failed to detect context type: %w", err)
	}

	// Override context if specified via flag
	if contextFlag != "" {
		ctx = parseContextFlag(contextFlag)
		if ctx == contextTypeUnknown {
			return fmt.Errorf("invalid context flag: %s (valid: span, log, metric, datapoint)", contextFlag)
		}
		// Re-parse data with forced context
		parsedData, err = parseDataWithContext(data, ctx)
		if err != nil {
			return fmt.Errorf("failed to parse data with context %s: %w", ctx, err)
		}
	}

	// 3. Apply OTTL transformation based on context
	err = applyTransformation(ottlStatement, ctx, parsedData)
	if err != nil {
		return fmt.Errorf("transformation failed: %w", err)
	}

	// 4. Output transformed data
	if err := outputTransformedData(ctx, parsedData); err != nil {
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

// readInputFile reads raw data from input file
func readInputFile(filename string) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("cannot open file %s: %w", filename, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %w", err)
	}

	return data, nil
}

// detectContextType automatically detects the data type and returns parsed data
func detectContextType(data []byte) (contextType, interface{}, error) {
	// Try traces first (backward compatibility)
	if traces, err := (&ptrace.JSONUnmarshaler{}).UnmarshalTraces(data); err == nil {
		if traces.ResourceSpans().Len() > 0 {
			return contextTypeSpan, traces, nil
		}
	}

	// Try logs
	if logs, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(data); err == nil {
		if logs.ResourceLogs().Len() > 0 {
			return contextTypeLog, logs, nil
		}
	}

	// Try metrics
	if metrics, err := (&pmetric.JSONUnmarshaler{}).UnmarshalMetrics(data); err == nil {
		if metrics.ResourceMetrics().Len() > 0 {
			return contextTypeMetric, metrics, nil
		}
	}

	return contextTypeUnknown, nil, fmt.Errorf("unable to detect data type from input")
}

// parseContextFlag converts string flag to contextType
func parseContextFlag(flag string) contextType {
	switch strings.ToLower(flag) {
	case "span":
		return contextTypeSpan
	case "log":
		return contextTypeLog
	case "metric":
		return contextTypeMetric
	case "datapoint":
		return contextTypeDatapoint
	default:
		return contextTypeUnknown
	}
}

// parseDataWithContext parses data with a specific context
func parseDataWithContext(data []byte, ctx contextType) (interface{}, error) {
	switch ctx {
	case contextTypeSpan:
		traces, err := (&ptrace.JSONUnmarshaler{}).UnmarshalTraces(data)
		if err != nil {
			return nil, fmt.Errorf("invalid OTLP traces JSON: %w", err)
		}
		return traces, nil
	case contextTypeLog:
		logs, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(data)
		if err != nil {
			return nil, fmt.Errorf("invalid OTLP logs JSON: %w", err)
		}
		return logs, nil
	case contextTypeMetric:
		metrics, err := (&pmetric.JSONUnmarshaler{}).UnmarshalMetrics(data)
		if err != nil {
			return nil, fmt.Errorf("invalid OTLP metrics JSON: %w", err)
		}
		return metrics, nil
	case contextTypeDatapoint:
		// For datapoint context, we need metrics data
		metrics, err := (&pmetric.JSONUnmarshaler{}).UnmarshalMetrics(data)
		if err != nil {
			return nil, fmt.Errorf("invalid OTLP metrics JSON for datapoint context: %w", err)
		}
		return metrics, nil
	default:
		return nil, fmt.Errorf("unsupported context type: %s", ctx)
	}
}


// applyTransformation applies OTTL statement based on context type
func applyTransformation(statement string, ctx contextType, data interface{}) error {
	switch ctx {
	case contextTypeSpan:
		traces, ok := data.(ptrace.Traces)
		if !ok {
			return fmt.Errorf("expected ptrace.Traces but got %T", data)
		}
		return applySpanTransformation(statement, traces)
	case contextTypeLog:
		logs, ok := data.(plog.Logs)
		if !ok {
			return fmt.Errorf("expected plog.Logs but got %T", data)
		}
		return applyLogTransformation(statement, logs)
	case contextTypeMetric:
		metrics, ok := data.(pmetric.Metrics)
		if !ok {
			return fmt.Errorf("expected pmetric.Metrics but got %T", data)
		}
		return applyMetricTransformation(statement, metrics)
	case contextTypeDatapoint:
		metrics, ok := data.(pmetric.Metrics)
		if !ok {
			return fmt.Errorf("expected pmetric.Metrics but got %T", data)
		}
		return applyDataPointTransformation(statement, metrics)
	default:
		return fmt.Errorf("unsupported context type: %s", ctx)
	}
}

// applySpanTransformation applies OTTL statement to traces (spans)
func applySpanTransformation(statement string, traces ptrace.Traces) error {
	parser, err := ottlspan.NewParser(ottlfuncs.StandardFuncs[ottlspan.TransformContext](), componenttest.NewNopTelemetrySettings())
	if err != nil {
		return fmt.Errorf("failed to create span parser: %w", err)
	}

	parsedStatement, err := parser.ParseStatement(statement)
	if err != nil {
		return fmt.Errorf("failed to parse span statement '%s': %w", statement, err)
	}

	resourceSpans := traces.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		rs := resourceSpans.At(i)
		scopeSpans := rs.ScopeSpans()

		for j := 0; j < scopeSpans.Len(); j++ {
			ss := scopeSpans.At(j)
			spans := ss.Spans()

			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				spanCtx := ottlspan.NewTransformContext(span, ss.Scope(), rs.Resource(), ss, rs)

				_, _, err := parsedStatement.Execute(context.Background(), spanCtx)
				if err != nil {
					return fmt.Errorf("failed to execute span transformation: %w", err)
				}
			}
		}
	}

	return nil
}

// applyLogTransformation applies OTTL statement to logs
func applyLogTransformation(statement string, logs plog.Logs) error {
	parser, err := ottllog.NewParser(ottlfuncs.StandardFuncs[ottllog.TransformContext](), componenttest.NewNopTelemetrySettings())
	if err != nil {
		return fmt.Errorf("failed to create log parser: %w", err)
	}

	parsedStatement, err := parser.ParseStatement(statement)
	if err != nil {
		return fmt.Errorf("failed to parse log statement '%s': %w", statement, err)
	}

	resourceLogs := logs.ResourceLogs()
	for i := 0; i < resourceLogs.Len(); i++ {
		rl := resourceLogs.At(i)
		scopeLogs := rl.ScopeLogs()

		for j := 0; j < scopeLogs.Len(); j++ {
			sl := scopeLogs.At(j)
			logRecords := sl.LogRecords()

			for k := 0; k < logRecords.Len(); k++ {
				logRecord := logRecords.At(k)
				logCtx := ottllog.NewTransformContext(logRecord, sl.Scope(), rl.Resource(), sl, rl)

				_, _, err := parsedStatement.Execute(context.Background(), logCtx)
				if err != nil {
					return fmt.Errorf("failed to execute log transformation: %w", err)
				}
			}
		}
	}

	return nil
}

// applyMetricTransformation applies OTTL statement to metrics
func applyMetricTransformation(statement string, metrics pmetric.Metrics) error {
	parser, err := ottlmetric.NewParser(ottlfuncs.StandardFuncs[ottlmetric.TransformContext](), componenttest.NewNopTelemetrySettings())
	if err != nil {
		return fmt.Errorf("failed to create metric parser: %w", err)
	}

	parsedStatement, err := parser.ParseStatement(statement)
	if err != nil {
		return fmt.Errorf("failed to parse metric statement '%s': %w", statement, err)
	}

	resourceMetrics := metrics.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)
		scopeMetrics := rm.ScopeMetrics()

		for j := 0; j < scopeMetrics.Len(); j++ {
			sm := scopeMetrics.At(j)
			metricSlice := sm.Metrics()

			for k := 0; k < metricSlice.Len(); k++ {
				metric := metricSlice.At(k)
				metricCtx := ottlmetric.NewTransformContext(metric, sm.Scope(), rm.Resource(), sm, rm)

				_, _, err := parsedStatement.Execute(context.Background(), metricCtx)
				if err != nil {
					return fmt.Errorf("failed to execute metric transformation: %w", err)
				}
			}
		}
	}

	return nil
}

// applyDataPointTransformation applies OTTL statement to metric data points
func applyDataPointTransformation(statement string, metrics pmetric.Metrics) error {
	parser, err := ottldatapoint.NewParser(ottlfuncs.StandardFuncs[ottldatapoint.TransformContext](), componenttest.NewNopTelemetrySettings())
	if err != nil {
		return fmt.Errorf("failed to create datapoint parser: %w", err)
	}

	parsedStatement, err := parser.ParseStatement(statement)
	if err != nil {
		return fmt.Errorf("failed to parse datapoint statement '%s': %w", statement, err)
	}

	resourceMetrics := metrics.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)
		scopeMetrics := rm.ScopeMetrics()

		for j := 0; j < scopeMetrics.Len(); j++ {
			sm := scopeMetrics.At(j)
			metricSlice := sm.Metrics()

			for k := 0; k < metricSlice.Len(); k++ {
				metric := metricSlice.At(k)

				// Apply to different metric types
				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					gauge := metric.Gauge()
					dataPoints := gauge.DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						dp := dataPoints.At(l)
						dpCtx := ottldatapoint.NewTransformContext(dp, metric, sm.Scope(), rm.Resource(), sm, rm)
						_, _, err := parsedStatement.Execute(context.Background(), dpCtx)
						if err != nil {
							return fmt.Errorf("failed to execute gauge datapoint transformation: %w", err)
						}
					}

				case pmetric.MetricTypeSum:
					sum := metric.Sum()
					dataPoints := sum.DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						dp := dataPoints.At(l)
						dpCtx := ottldatapoint.NewTransformContext(dp, metric, sm.Scope(), rm.Resource(), sm, rm)
						_, _, err := parsedStatement.Execute(context.Background(), dpCtx)
						if err != nil {
							return fmt.Errorf("failed to execute sum datapoint transformation: %w", err)
						}
					}

				case pmetric.MetricTypeHistogram:
					histogram := metric.Histogram()
					dataPoints := histogram.DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						dp := dataPoints.At(l)
						dpCtx := ottldatapoint.NewTransformContext(dp, metric, sm.Scope(), rm.Resource(), sm, rm)
						_, _, err := parsedStatement.Execute(context.Background(), dpCtx)
						if err != nil {
							return fmt.Errorf("failed to execute histogram datapoint transformation: %w", err)
						}
					}

				case pmetric.MetricTypeExponentialHistogram:
					expHistogram := metric.ExponentialHistogram()
					dataPoints := expHistogram.DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						dp := dataPoints.At(l)
						dpCtx := ottldatapoint.NewTransformContext(dp, metric, sm.Scope(), rm.Resource(), sm, rm)
						_, _, err := parsedStatement.Execute(context.Background(), dpCtx)
						if err != nil {
							return fmt.Errorf("failed to execute exponential histogram datapoint transformation: %w", err)
						}
					}

				case pmetric.MetricTypeSummary:
					summary := metric.Summary()
					dataPoints := summary.DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						dp := dataPoints.At(l)
						dpCtx := ottldatapoint.NewTransformContext(dp, metric, sm.Scope(), rm.Resource(), sm, rm)
						_, _, err := parsedStatement.Execute(context.Background(), dpCtx)
						if err != nil {
							return fmt.Errorf("failed to execute summary datapoint transformation: %w", err)
						}
					}
				}
			}
		}
	}

	return nil
}

// outputTransformedData outputs data as JSON based on context type
func outputTransformedData(ctx contextType, data interface{}) error {
	switch ctx {
	case contextTypeSpan:
		traces, ok := data.(ptrace.Traces)
		if !ok {
			return fmt.Errorf("expected ptrace.Traces but got %T", data)
		}
		return outputTransformedTraces(traces)
	case contextTypeLog:
		logs, ok := data.(plog.Logs)
		if !ok {
			return fmt.Errorf("expected plog.Logs but got %T", data)
		}
		return outputTransformedLogs(logs)
	case contextTypeMetric, contextTypeDatapoint:
		metrics, ok := data.(pmetric.Metrics)
		if !ok {
			return fmt.Errorf("expected pmetric.Metrics but got %T", data)
		}
		return outputTransformedMetrics(metrics)
	default:
		return fmt.Errorf("unsupported context type: %s", ctx)
	}
}

// outputTransformedTraces outputs traces as JSON using pdata marshaler
func outputTransformedTraces(traces ptrace.Traces) error {
	marshaler := &ptrace.JSONMarshaler{}
	jsonData, err := marshaler.MarshalTraces(traces)
	if err != nil {
		return fmt.Errorf("failed to marshal traces to JSON: %w", err)
	}
	fmt.Print(string(jsonData))
	return nil
}

// outputTransformedLogs outputs logs as JSON using pdata marshaler
func outputTransformedLogs(logs plog.Logs) error {
	marshaler := &plog.JSONMarshaler{}
	jsonData, err := marshaler.MarshalLogs(logs)
	if err != nil {
		return fmt.Errorf("failed to marshal logs to JSON: %w", err)
	}
	fmt.Print(string(jsonData))
	return nil
}

// outputTransformedMetrics outputs metrics as JSON using pdata marshaler
func outputTransformedMetrics(metrics pmetric.Metrics) error {
	marshaler := &pmetric.JSONMarshaler{}
	jsonData, err := marshaler.MarshalMetrics(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics to JSON: %w", err)
	}
	fmt.Print(string(jsonData))
	return nil
}
