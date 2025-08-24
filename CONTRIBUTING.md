# Contributing to ottl-cli

## 🤖 Project Philosophy: Agent-First Development

**This project serves as an experimental playground for AI-assisted development.** We are exploring the
boundaries of what's possible when coding agents handle the majority of development tasks. Manual coding
should be avoided whenever possible in favor of agent-driven development.

### Why Agent-First?

- **Learning**: Understanding the capabilities and limitations of coding agents
- **Efficiency**: Testing if agents can maintain and evolve production-quality code
- **Innovation**: Discovering new patterns for human-AI collaboration
- **Quality**: Evaluating if agent-written code meets or exceeds human standards

## 📋 Contribution Rules

### 1. Conventional Commits (Required)

All commits and PR titles MUST follow the
[Conventional Commits](https://www.conventionalcommits.org/) specification:

```text
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types:**

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, semicolons, etc.)
- `refactor`: Code refactoring without changing functionality
- `test`: Adding or updating tests
- `chore`: Maintenance tasks, dependency updates
- `perf`: Performance improvements
- `ci`: CI/CD configuration changes

**Examples:**

```bash
feat: add metric context support for OTTL transformations
fix(parser): handle empty input gracefully
docs: update README with new examples
chore(deps): bump opentelemetry to v1.30.0
```

### 2. Agent-First Development Process

When contributing to this project:

1. **Use AI Agents First**: Before writing any code manually, attempt to use AI coding agents
   (Claude, GitHub Copilot, Cursor, etc.)

2. **Document Agent Usage**: Feel free to share in your PR description:
   - Which agent(s) you used
   - What prompts/instructions worked well
   - Any limitations you encountered
   - Percentage of code written by agents vs. manual

3. **Share Learning**: If an agent couldn't complete a task, consider documenting:
   - What was attempted
   - Why it failed
   - What manual intervention was needed
   - Suggestions for better agent prompting

### 3. Code Quality Standards

Even though agents write most code, all contributions must meet these standards:

- **Simplicity**: Follow the ultra-lean philosophy - avoid unnecessary complexity
- **Single File**: Maintain the single-file architecture unless absolutely necessary
- **Official Packages**: Use only official OpenTelemetry packages
- **Performance**: Ensure <50ms startup and <100ms transformation times
- **Testing**: Include tests for new functionality
- **Documentation**: Update README for user-facing changes

### 4. Pull Request Process

1. **Create an Issue First**: Describe what you want to achieve
2. **Branch Naming**: Use conventional format: `feat/`, `fix/`, `docs/`, etc.
3. **Small PRs**: Keep changes focused and reviewable
4. **CI Must Pass**: All GitHub Actions workflows must succeed
5. **Agent Review**: Our Claude Code Review bot will automatically review your PR

### 5. Testing Requirements

- Run all tests: `go test ./...`
- Format code: `go fmt ./...`
- Test with example: `make example`
- Verify build: `make build`
- Check linting: `golangci-lint run` (if installed)

## 🤝 How to Contribute

### For Agent Experiments

We especially welcome contributions that:

1. **Push Agent Boundaries**: Try complex refactoring or feature additions
2. **Improve Agent Prompts**: Share effective prompting strategies in `CLAUDE.md`
3. **Document Patterns**: Identify repeatable patterns for agent success
4. **Benchmark Performance**: Compare agent vs. human code quality/speed

## 🚀 Getting Started

1. Fork the repository
2. Create a feature branch following naming conventions
3. Use your preferred AI agent to implement changes
4. Ensure all tests pass
5. Submit a PR (optionally share your agent experience)

## 🔬 Experimental Features

Feel free to propose experimental features that test agent capabilities:

- Complex refactoring tasks
- Performance optimizations
- Architecture decisions
- Test generation
- Documentation automation

## 📝 Improving Agent Instructions

If you discover better ways to instruct agents for this project, please update:

- `CLAUDE.md`: Project-specific agent instructions
- This file: Share general agent strategies
- PR descriptions: Document specific techniques

## ❓ Questions?

If you're unsure about anything:

1. Check existing issues and PRs for similar questions
2. Open a discussion issue
3. Tag it with `question` and `agent-experiment`

## 🎯 Project Goals

Remember, this project aims to:

1. Provide a useful OTTL CLI tool (primary)
2. Explore agent-driven development (experimental)
3. Document learnings for the community
4. Maintain high code quality despite automation

---

**Note**: This is a living experiment. We encourage creative approaches to agent-driven development
while maintaining production-quality standards. Your contributions help us understand the future of
AI-assisted software development.

## 📜 License

By contributing, you agree that your contributions will be licensed under the same license as this
project (Apache 2.0).
