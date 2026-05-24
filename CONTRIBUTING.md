# Contributing to AgentX

Thank you for your interest in contributing to AgentX! This project is a community-driven effort to build the lightweight and versatile personal AI assistant. We welcome contributions of all kinds: bug fixes, features, documentation, translations, and testing.

AgentX itself was substantially developed with AI assistance — we embrace this approach and have built our contribution process around it.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Ways to Contribute](#ways-to-contribute)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [AI-Assisted Contributions](#ai-assisted-contributions)
- [Pull Request Process](#pull-request-process)
- [Branch Strategy](#branch-strategy)
- [Code Review](#code-review)
- [Communication](#communication)

---

## Code of Conduct

We are committed to maintaining a welcoming and respectful community. Be kind, constructive, and assume good faith. Harassment or discrimination of any kind will not be tolerated.

---

## Ways to Contribute

- **Bug reports** — Open an issue using the bug report template.
- **Feature requests** — Open an issue using the feature request template; discuss before implementing.
- **Code** — Fix bugs or implement features. See the workflow below.
- **Documentation** — Improve READMEs, docs, inline comments, or translations.
- **Testing** — Run AgentX on new hardware, channels, or LLM providers and report your results.

For substantial new features, please open an issue first to discuss the design before writing code. This prevents wasted effort and ensures alignment with the project's direction.

---

## Getting Started

1. **Fork** the repository on GitHub.
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/<your-username>/agentx.git
   cd agentx
   ```
3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/Agentx-network/agentx.git
   ```

---

## Development Setup

### Prerequisites

- Go 1.25 or later
- `make`

### Build

```bash
make build       # Build binary (runs go generate first)
make generate    # Run go generate only
make check       # Full pre-commit check: deps + fmt + vet + test
```

### Running Tests

```bash
make test                                    # Run all tests
go test -run TestName -v ./pkg/session/      # Run a single test
go test -bench=. -benchmem -run='^$' ./...  # Run benchmarks
```

### Code Style

```bash
make fmt   # Format code
make vet   # Static analysis
make lint  # Full linter run
```

All CI checks must pass before a PR can be merged. Run `make check` locally before pushing to catch issues early.

---

## Making Changes

### Branching

Always branch off `main` and target `main` in your PR. Never push directly to `main` or any `release/*` branch:

```bash
git checkout main
git pull upstream main
git checkout -b your-feature-branch
```

Use descriptive branch names, e.g. `fix/telegram-timeout`, `feat/ollama-provider`, `docs/contributing-guide`.

### Commits

- Write clear, concise commit messages in English.
- Use the imperative mood: "Add retry logic" not "Added retry logic".
- Reference the related issue when relevant: `Fix session leak (#123)`.
- Keep commits focused. One logical change per commit is preferred.
- For minor cleanups or typo fixes, squash them into a single commit before opening a PR.
- Refer to https://www.conventionalcommits.org/zh-hans/v1.0.0/

### Keeping Up to Date

Rebase your branch onto upstream `main` before opening a PR:

```bash
git fetch upstream
git rebase upstream/main
```

---

## AI-Assisted Contributions

AgentX was built with substantial AI assistance, and we fully embrace AI-assisted development. However, contributors must understand their responsibilities when using AI tools.

### Disclosure Is Required

Every PR must disclose AI involvement using the PR template's **🤖 AI Code Generation** section. There are three levels:

| Level | Description |
|---|---|
| 🤖 Fully AI-generated | AI wrote the code; contributor reviewed and validated it |
| 🛠️ Mostly AI-generated | AI produced the draft; contributor made significant modifications |
| 👨‍💻 Mostly Human-written | Contributor led; AI provided suggestions or none at all |

Honest disclosure is expected. There is no stigma attached to any level — what matters is the quality of the contribution.

### You Are Responsible for What You Submit

Using AI to generate code does not reduce your responsibility as the contributor. Before opening a PR with AI-generated code, you must:

- **Read and understand** every line of the generated code.
- **Test it** in a real environment (see the Test Environment section of the PR template).
- **Check for security issues** — AI models can generate subtly insecure code (e.g., path traversal, injection, credential exposure). Review carefully.
- **Verify correctness** — AI-generated logic can be plausible-sounding but wrong. Validate the behavior, not just the syntax.

PRs where it is clear the contributor has not read or tested the AI-generated code will be closed without review.

### AI-Generated Code Quality Standards

AI-generated contributions are held to the **same quality bar** as human-written code:

- It must pass all CI checks (`make check`).
- It must be idiomatic Go and consistent with the existing codebase style.
- It must not introduce unnecessary abstractions, dead code, or over-engineering.
- It must include or update tests where appropriate.

### Security Review

AI-generated code requires extra security scrutiny. Pay special attention to:

- File path handling and sandbox escapes (see commit `244eb0b` for a real example)
- External input validation in channel handlers and tool implementations
- Credential or secret handling
- Command execution (`exec.Command`, shell invocations)

If you are unsure whether a piece of AI-generated code is safe, say so in the PR — reviewers will help.

---

## Pull Request Process

### Before Opening a PR

- [ ] Run `make check` and ensure it passes locally.
- [ ] Fill in the PR template completely, including the AI disclosure section.
- [ ] Link any related issue(s) in the PR description.
- [ ] Keep the PR focused. Avoid bundling unrelated changes together.

### PR Template Sections

The PR template asks for:

- **Description** — What does this change do and why?
- **Type of Change** — Bug fix, feature, docs, or refactor.
- **AI Code Generation** — Disclosure of AI involvement (required).
- **Related Issue** — Link to the issue this addresses.
- **Technical Context** — Reference URLs and reasoning (skip for pure docs PRs).
- **Test Environment** — Hardware, OS, model/provider, and channels used for testing.
- **Evidence** — Optional logs or screenshots demonstrating the change works.
- **Checklist** — Self-review confirmation.

### PR Size

Prefer small, reviewable PRs. A PR that changes 200 lines across 5 files is much easier to review than one that changes 2000 lines across 30 files. If your feature is large, consider splitting it into a series of smaller, logically complete PRs.

---

## Branch Strategy

### Long-Lived Branches

- **`main`** — the active development branch. All feature PRs target `main`. The branch is protected: direct pushes are not permitted, and at least one maintainer approval is required before merging.
- **`release/x.y`** — stable release branches, cut from `main` when a version is ready to ship. These branches are more strictly protected than `main`.

### Requirements to Merge into `main`

A PR can only be merged when all of the following are satisfied:

1. **CI passes** — All GitHub Actions workflows (lint, test, build) must be green.
2. **Reviewer approval** — At least one maintainer has approved the PR.
3. **No unresolved review comments** — All review threads must be resolved.
4. **PR template is complete** — Including AI disclosure and test environment.

### Who Can Merge

Only maintainers can merge PRs. Contributors cannot merge their own PRs, even if they have write access.

### Merge Strategy

We use **squash merge** for most PRs to keep the `main` history clean and readable. Each merged PR becomes a single commit referencing the PR number, e.g.:

```
feat: Add Ollama provider support (#491)
```

If a PR consists of multiple independent, well-separated commits that tell a clear story, a regular merge may be used at the maintainer's discretion.

### Release Branches

When a version is ready, maintainers cut a `release/x.y` branch from `main`. After that point:

- **New features are not backported.** The release branch receives no new functionality after it is cut.
- **Security fixes and critical bug fixes are cherry-picked.** If a fix in `main` qualifies (security vulnerability, data loss, crash), maintainers will cherry-pick the relevant commit(s) onto the affected `release/x.y` branch and issue a patch release.

If you believe a fix in `main` should be backported to a release branch, note it in the PR description or open a separate issue. The decision rests with the maintainers.

Release branches have stricter protections than `main` and are never directly pushed to under any circumstances.

---

## Code Review

### For Contributors

- Respond to review comments within a reasonable time. If you need more time, say so.
- When you update a PR in response to feedback, briefly note what changed (e.g., "Updated to use `sync.RWMutex` as suggested").
- If you disagree with feedback, engage respectfully. Explain your reasoning; reviewers can be wrong too.
- Do not force-push after a review has started — it makes it harder for reviewers to see what changed. Use additional commits instead; the maintainer will squash on merge.

### For Reviewers

Review for:

1. **Correctness** — Does the code do what it claims? Are there edge cases?
2. **Security** — Especially for AI-generated code, tool implementations, and channel handlers.
3. **Architecture** — Is the approach consistent with the existing design?
4. **Simplicity** — Is there a simpler solution? Does this add unnecessary complexity?
5. **Tests** — Are the changes covered by tests? Are existing tests still meaningful?

Be constructive and specific. "This could have a race condition if two goroutines call this concurrently — consider using a mutex here" is better than "this looks wrong".


### Reviewer List
Once your PR is submitted, you can reach out to the assigned reviewers listed in the following table.

|Function| Reviewer|
|---     |---      |
|Provider|@yinwm   |
|Channel |@yinwm   |
|Agent   |@lxowalle|
|Tools   |@lxowalle|
|SKill   ||
|MCP     ||
|Optimization|@lxowalle|
|Security||
|AI CI   |@imguoguo|
|UX      ||
|Document||

---

## Releasing

### How Releases Work

Releases publish pre-built binaries to GitHub Releases. The desktop app (`agentx-desktop`) downloads the **raw binary** (not the archive) from the latest release using the URL pattern:

```
https://github.com/Agentx-network/agentx/releases/latest/download/agentx-{os}-{arch}
```

This means every release **must include raw binaries** alongside the `.tar.gz`/`.zip` archives.

### Release Asset Naming

| Asset | Description |
| --- | --- |
| `agentx-linux-amd64` | Raw binary — Linux x86_64 |
| `agentx-linux-arm64` | Raw binary — Linux ARM64 (also used by Android/Termux) |
| `agentx-linux-armv7` | Raw binary — Linux ARMv7 |
| `agentx-darwin-amd64` | Raw binary — macOS Intel |
| `agentx-darwin-arm64` | Raw binary — macOS Apple Silicon |
| `agentx-windows-amd64.exe` | Raw binary — Windows x86_64 |
| `agentx-linux-amd64.tar.gz` | Archive — for manual download |
| `agentx-windows-amd64.zip` | Archive — for manual download |
| `checksums.txt` | SHA-256 checksums for all assets |

### Creating a Release

#### Option 1: GitHub Actions (preferred)

Trigger the release workflow from the Actions tab or CLI:

```bash
gh workflow run release.yml --field tag=v0.5.0 --field prerelease=false --field draft=false
```

This creates the tag, builds all platforms via GoReleaser, and publishes the release.

**Prerequisites**: The following secrets/variables must be set in the repository:
- `DOCKERHUB_USERNAME` — Docker Hub username
- `DOCKERHUB_TOKEN` — Docker Hub access token
- `DOCKERHUB_REPOSITORY` (variable) — e.g. `agentxnetwork/agentx`

#### Option 2: Local Build + Manual Upload

If CI fails or you need a quick release, build locally and upload:

```bash
# 1. Set build variables
VERSION=0.5.0
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
GOVERSION=$(go version | awk '{print $3}')
LDFLAGS="-s -w \
  -X github.com/Agentx-network/agentx/cmd/agentx/internal.version=${VERSION} \
  -X github.com/Agentx-network/agentx/cmd/agentx/internal.gitCommit=${COMMIT} \
  -X github.com/Agentx-network/agentx/cmd/agentx/internal.buildTime=${DATE} \
  -X github.com/Agentx-network/agentx/cmd/agentx/internal.goVersion=${GOVERSION}"

# 2. Build all platforms
mkdir -p dist
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64       go build -tags stdjson -ldflags "$LDFLAGS" -o dist/agentx-linux-amd64       ./cmd/agentx
CGO_ENABLED=0 GOOS=linux   GOARCH=arm64       go build -tags stdjson -ldflags "$LDFLAGS" -o dist/agentx-linux-arm64       ./cmd/agentx
CGO_ENABLED=0 GOOS=linux   GOARCH=arm GOARM=7 go build -tags stdjson -ldflags "$LDFLAGS" -o dist/agentx-linux-armv7       ./cmd/agentx
CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64       go build -tags stdjson -ldflags "$LDFLAGS" -o dist/agentx-darwin-amd64      ./cmd/agentx
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64       go build -tags stdjson -ldflags "$LDFLAGS" -o dist/agentx-darwin-arm64      ./cmd/agentx
CGO_ENABLED=0 GOOS=windows GOARCH=amd64       go build -tags stdjson -ldflags "$LDFLAGS" -o dist/agentx-windows-amd64.exe ./cmd/agentx

# 3. Create archives (optional, for manual downloaders)
cd dist
for f in agentx-linux-* agentx-darwin-* agentx-freebsd-*; do
  [ -f "$f" ] && tar czf "${f}.tar.gz" "$f"
done
[ -f agentx-windows-amd64.exe ] && zip -q agentx-windows-amd64.zip agentx-windows-amd64.exe
sha256sum * > checksums.txt
cd ..

# 4. Create tag and release
git tag -a v${VERSION} -m "Release v${VERSION}"
git push origin v${VERSION}
gh release create v${VERSION} --title "v${VERSION}" --generate-notes \
  dist/agentx-linux-amd64 \
  dist/agentx-linux-arm64 \
  dist/agentx-linux-armv7 \
  dist/agentx-darwin-amd64 \
  dist/agentx-darwin-arm64 \
  dist/agentx-windows-amd64.exe \
  dist/*.tar.gz \
  dist/*.zip \
  dist/checksums.txt
```

### Important Notes

- **Raw binaries are required** — the desktop app and `install.sh` download raw binaries, not archives. If you only upload `.tar.gz` files, the installer will fail with a 404.
- **Arch naming** — Go's `runtime.GOARCH` returns `arm`, but the binary is named `armv7` for clarity. The desktop installer maps this automatically.
- The `latest` redirect (`/releases/latest/download/...`) always points to the most recent non-prerelease, non-draft release.

---

## Communication

- **GitHub Issues** — Bug reports, feature requests, design discussions.
- **GitHub Discussions** — General questions, ideas, community conversation.
- **Pull Request comments** — Code-specific feedback.
- **Wechat&Discord** — We will invite you when you have at least one merged PR

When in doubt, open an issue before writing code. It costs little and prevents wasted effort.

---

## A Note on the Project's AI-Driven Origin

AgentX's architecture was substantially designed and implemented with AI assistance, guided by human oversight. If you find something that looks odd or over-engineered, it may be an artifact of that process — opening an issue to discuss it is always welcome.

We believe AI-assisted development done responsibly produces great results. We also believe humans must remain accountable for what they ship. These two beliefs are not in conflict.

Thank you for contributing!
