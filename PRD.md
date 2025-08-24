# Product Requirements Document: ottl-cli

## Executive Summary

**Product Name:** ottl-cli  
**Version:** 1.0.0 (MVP)  
**Document Status:** Final  
**Last Updated:** 2025-08-17

### Problem Statement

OpenTelemetry practitioners need a lightweight, command-line tool to test and validate OTTL
(OpenTelemetry Transformation Language) statements on telemetry data before deploying them to
production collectors. Current testing methods require full collector deployments or manual
validation, leading to slow iteration cycles and potential production issues.

### Solution Overview

ottl-cli is a lean command-line interface that wraps the official OpenTelemetry OTTL library,
enabling developers to quickly test transformations on sample data with immediate feedback. The tool
follows Unix philosophy: do one thing well, with composable commands that integrate into existing
workflows.

## Business Objectives

### Primary Goals

1. **Reduce Development Cycle Time**: Enable sub-second testing of OTTL statements without collector deployment
2. **Improve Transformation Quality**: Catch errors before production deployment
3. **Lower Barrier to Entry**: Provide simple, focused tooling for OTTL adoption
4. **Maintain Compatibility**: Ensure 100% alignment with official OTTL library behavior

### Success Criteria

- **Adoption Metrics**:
  - 1,000+ downloads within 3 months of release
  - Integration into at least 5 CI/CD pipelines
  - Positive feedback from OpenTelemetry community

- **Technical Metrics**:
  - Zero discrepancies with official OTTL library behavior
  - Sub-100ms execution time for typical transformations
  - 100% JSON parsing success rate for valid OTLP data

- **Quality Metrics**:
  - <1% crash rate in production usage
  - Clear error messages leading to <5 minute mean time to resolution
  - Zero security vulnerabilities in dependencies

## User Stories

### Primary User: DevOps Engineer

**As a** DevOps engineer managing OpenTelemetry collectors  
**I want to** test OTTL transformations on sample data  
**So that** I can validate transformations before deploying to production

**Acceptance Criteria:**

- Can pipe OTTL statement via stdin
- Can specify input data file via command flag
- Receives transformed JSON output on stdout
- Gets clear error messages when transformation fails

### Primary User: SRE

**As an** SRE debugging telemetry issues  
**I want to** quickly test OTTL statements on production data samples  
**So that** I can identify and fix transformation problems

**Acceptance Criteria:**

- Can test single statements without configuration files
- Output is valid JSON that can be piped to other tools
- Error messages include actionable remediation steps
- No extraneous output pollutes stdout

### Secondary User: Developer

**As a** developer learning OTTL  
**I want to** experiment with transformations interactively  
**So that** I can understand OTTL syntax and capabilities

**Acceptance Criteria:**

- Simple command structure with minimal flags
- Predictable input/output behavior
- Examples provided in documentation
- Fast feedback loop (<1 second)

## Functional Requirements

### FR1: Transform Command (MVP)

**ID:** FR1  
**Priority:** P0 (Required for MVP)  
**Description:** Execute OTTL transformation on input data

**Specifications:**

- Command: `ottl-cli transform --input-file <path> < statement.ottl`
- Input data format: JSON (OTLP format)
- Statement input: Via stdin (single statement)
- Output format: JSON to stdout
- Context support: Span context only (initial release)

**Example Usage:**

```bash
echo 'set(attributes["env"], "production")' | ottl-cli transform --input-file trace.json
```

### FR2: Input File Processing

**ID:** FR2  
**Priority:** P0 (Required for MVP)  
**Description:** Read and parse OTLP JSON input files

**Specifications:**

- Support standard OTLP JSON format for traces
- File path specified via `--input-file` flag
- Validate JSON structure before processing
- Support files up to 100MB (initial limit)

### FR3: OTTL Statement Execution

**ID:** FR3  
**Priority:** P0 (Required for MVP)  
**Description:** Parse and execute OTTL statements using official library

**Specifications:**

- Use github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl
- Support all standard OTTL functions for span context
- Maintain 100% compatibility with collector behavior
- Single statement per invocation

### FR4: Output Generation

**ID:** FR4  
**Priority:** P0 (Required for MVP)  
**Description:** Generate transformed JSON output

**Specifications:**

- Output valid OTLP JSON format
- Write only to stdout (no file output in MVP)
- Preserve structure of input data
- Include all spans (transformed and untransformed)

### FR5: Error Handling

**ID:** FR5  
**Priority:** P0 (Required for MVP)  
**Description:** Provide clear error messages with remediation

**Specifications:**

- Error format: `Error: <message>\nRemediation: <suggestion>`
- No output to stdout on error (only stderr)
- Exit code 1 on any error
- Categories:
  - Invalid JSON input
  - OTTL syntax errors
  - Runtime transformation errors
  - File access errors

### FR6: Validate Command (Future)

**ID:** FR6  
**Priority:** P2 (Post-MVP)  
**Description:** Validate OTTL statements without execution

**Specifications:**

- Command: `ottl-cli validate --input-file <path> < statement.ottl`
- Syntax validation only (no execution)
- Report all syntax errors
- Exit code 0 if valid, 1 if invalid

## Non-Functional Requirements

### NFR1: Performance

**ID:** NFR1  
**Priority:** P1  
**Description:** Tool execution performance requirements

**Specifications:**

- Startup time: <50ms
- Transformation execution: <100ms for 1MB input file
- Memory usage: <100MB for typical operations
- Binary size: <20MB maximum

### NFR2: Reliability

**ID:** NFR2  
**Priority:** P0  
**Description:** System stability and error recovery

**Specifications:**

- Graceful handling of malformed input
- No panics on invalid data
- Clean exit on all error conditions
- No resource leaks

### NFR3: Usability

**ID:** NFR3  
**Priority:** P0  
**Description:** User experience requirements

**Specifications:**

- Single binary distribution
- No configuration files required
- Standard Unix command patterns
- Composable with shell pipelines

### NFR4: Compatibility

**ID:** NFR4  
**Priority:** P0  
**Description:** System and format compatibility

**Specifications:**

- Linux support (amd64, arm64)
- Go 1.24+ for building
- OTLP JSON format compatibility
- UTF-8 encoding support

### NFR5: Security

**ID:** NFR5  
**Priority:** P1  
**Description:** Security requirements

**Specifications:**

- No network access required
- File system access limited to specified input file
- No execution of arbitrary code
- Dependency scanning in CI/CD

### NFR6: Maintainability

**ID:** NFR6  
**Priority:** P1  
**Description:** Code quality and maintenance requirements

**Specifications:**

- Ultra-lean single-file implementation (target: 175 lines)
- Direct usage of official OpenTelemetry packages
- Integration tests for core workflows
- Automated releases via GoReleaser

## User Experience Flow

### Primary Flow: Transform Telemetry Data

1. **Prepare Input Data**
   - User has OTLP JSON file (e.g., trace.json)
   - File contains valid telemetry data

2. **Write OTTL Statement**
   - User creates OTTL statement
   - Can be in file or as string

3. **Execute Transformation**

   ```bash
   echo 'set(attributes["env"], "prod")' | ottl-cli transform --input-file trace.json
   ```

4. **Process Output**
   - Success: Transformed JSON printed to stdout
   - Failure: Error message to stderr, exit code 1

5. **Integration Options**
   - Pipe output to jq for further processing
   - Redirect to file for storage
   - Use in CI/CD validation scripts

### Error Flow: Invalid Statement

1. **User provides invalid OTTL syntax**

   ```bash
   echo 'set(invalid syntax)' | ottl-cli transform --input-file trace.json
   ```

2. **Tool detects syntax error**

3. **Error output to stderr**

   ```text
   Error: Invalid OTTL syntax at position 4: expected '(' after function name
   Remediation: Check OTTL function syntax. Example: set(attributes["key"], "value")
   ```

4. **Exit with code 1**

## Technical Considerations

### Architecture

- **Pattern**: Ultra-lean single-file application (target: 175 lines)
- **Structure**: Direct implementation with minimal abstraction
  - Single main.go file with clear function separation
  - Direct usage of official OpenTelemetry packages
  - No complex layering or domain abstractions

### Dependencies

- **Target Dependencies** (maximum 4):
  - github.com/spf13/cobra (CLI framework)
  - github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl (OTTL library)
  - go.opentelemetry.io/collector/pdata (OTLP data structures)
  - go.opentelemetry.io/collector/component (Testing utilities)

### Build and Distribution

- **Build Tool**: Go modules
- **Distribution**: GoReleaser
- **Artifacts**:
  - Linux binaries (amd64, arm64)
  - Docker images
  - Checksums and signatures

### Testing Strategy

- **Unit Tests**: Domain and application logic
- **Integration Tests**: OTTL library integration
- **E2E Tests**: Full command execution flows
- **Test Data**: Sample OTLP JSON files in local/payload-examples/

## Dependencies and Integration Points

### External Dependencies

1. **OTTL Library**
   - Version: Latest stable
   - Update strategy: Monthly dependency updates
   - Breaking change monitoring

2. **Go Runtime**
   - Minimum version: 1.24
   - Build environment: Linux/Docker

### Integration Points

1. **Shell/Terminal**
   - Standard Unix pipes
   - Exit codes
   - Signal handling (SIGINT, SIGTERM)

2. **File System**
   - Read-only access to input files
   - Current directory resolution
   - Path validation

3. **CI/CD Systems**
   - GitHub Actions integration
   - Exit code-based validation
   - JSON output parsing

## Success Metrics and KPIs

### Launch Metrics (Month 1)

- Successfully process 100% of valid OTLP JSON files
- Zero crashes in normal operation
- Documentation coverage for all commands

### Growth Metrics (Month 3)

- 1,000+ unique downloads
- 5+ GitHub stars
- 3+ community contributions

### Quality Metrics (Ongoing)

- Mean time to resolution for errors: <5 minutes
- User-reported bugs: <5 per month
- Response time for issues: <48 hours

## Risk Assessment

### Technical Risks

#### Risk 1: OTTL Library Breaking Changes

- **Probability**: Medium
- **Impact**: High
- **Mitigation**: Pin specific versions, comprehensive test suite, monitor upstream changes

#### Risk 2: Large File Performance

- **Probability**: Low
- **Impact**: Medium
- **Mitigation**: Document file size limits, implement streaming if needed post-MVP

#### Risk 3: JSON Format Variations

- **Probability**: Medium
- **Impact**: Medium
- **Mitigation**: Strict OTLP format validation, clear error messages for unsupported formats

### Business Risks

#### Risk 4: Low Adoption

- **Probability**: Medium
- **Impact**: High
- **Mitigation**: Community engagement, documentation, integration examples

#### Risk 5: Scope Creep

- **Probability**: High
- **Impact**: Medium
- **Mitigation**: Strict MVP scope, clear roadmap for future features

## Scope Definition

### In Scope for MVP

- Transform command for span context
- JSON input via file flag
- OTTL statement via stdin
- JSON output to stdout
- Comprehensive error messages with clear diagnostics
- Linux binary distribution
- Complete OTTL function library support
- Production-ready performance and reliability

### Out of Scope (Current Version)

- Multiple OTTL statements per invocation
- Configuration files
- Network input/output
- GUI or web interface
- Windows/macOS binaries (future consideration)
- Metrics and logs contexts
- Batch processing
- Statement validation without execution

### Future Considerations

- Additional telemetry contexts
- Statement composition
- Performance profiling
- Cloud-native integrations
- Interactive mode
- Visual transformation preview

## Appendix

### A. Example OTTL Statements

```ottl
set(attributes["environment"], "production")
delete(attributes["debug"])
replace_pattern(name, "^/api/v1", "/api/v2")
set(status.code, 1) where attributes["error"] == "true"
```

### B. Sample Input/Output

**Input (trace.json):**

```json
{
  "resourceSpans": [{
    "scopeSpans": [{
      "spans": [{
        "name": "GET /api/users",
        "attributes": [{
          "key": "http.method",
          "value": {"stringValue": "GET"}
        }]
      }]
    }]
  }]
}
```

**Statement:**

```ottl
set(attributes["environment"], "prod")
```

**Output:**

```json
{
  "resourceSpans": [{
    "scopeSpans": [{
      "spans": [{
        "name": "GET /api/users",
        "attributes": [
          {
            "key": "http.method",
            "value": {"stringValue": "GET"}
          },
          {
            "key": "environment",
            "value": {"stringValue": "prod"}
          }
        ]
      }]
    }]
  }]
}
```

### C. Error Message Examples

**Invalid JSON:**

```text
Error: Invalid JSON input at line 5, column 12: unexpected token '}'
Remediation: Validate JSON syntax using a JSON validator or jq
```

**OTTL Syntax Error:**

```text
Error: Invalid OTTL syntax: unknown function 'sets' at position 0
Remediation: Did you mean 'set'? Check available functions in OTTL documentation
```

**File Not Found:**

```text
Error: Cannot read input file: trace.json: no such file or directory
Remediation: Verify file path and permissions. Use absolute path if needed
```

---

**Document Version:** 1.0.0  
**Status:** Requirements Specification  
**Target Implementation:** Ultra-lean single-file approach (175 lines)  
**Next Review:** Post-Implementation
