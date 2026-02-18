---
name: release
description: Create a new release by tagging and pushing a version. Guides through changelog review, version selection, and tag push which triggers goreleaser CI.
argument-hint: "[version]"
disable-model-invocation: true
allowed-tools: Bash(git *), Bash(gh *), Read, Grep, Glob
---

Create a release for personal-vault. Version argument (if provided): $ARGUMENTS

## Steps

1. **Check prerequisites**
   - Confirm on `main` branch with clean working tree (`git status`)
   - Confirm all changes are pushed (`git log origin/main..HEAD`)

2. **Find previous release**
   - Run `git tag --sort=-v:refname | head -5` to find the latest version tag
   - If no tags exist, this is the first release (v0.1.0)

3. **Show changelog since last release**
   - Run `git log <last-tag>..HEAD --oneline` (or `git log --oneline -20` if first release)
   - Summarize the changes in a few bullet points for the user

4. **Determine version**
   - If `$ARGUMENTS` was provided and starts with `v`, use it directly
   - Otherwise, suggest the next version based on the changes:
     - **patch** (v0.1.x): bug fixes, docs, internal changes
     - **minor** (v0.x.0): new features, new commands, new config
     - **major** (vx.0.0): breaking changes
   - Ask the user to confirm or choose a different version

5. **Run tests**
   - Run `go test -race ./...` and confirm they pass
   - Do NOT proceed if tests fail

6. **Create and push tag**
   - Run `git tag -a <version> -m "Release <version>"`
   - Ask the user for confirmation before pushing
   - Run `git push origin <version>`

7. **Confirm**
   - Print the GitHub Actions URL: `https://github.com/lovincyrus/personal-vault/actions`
   - Print the releases URL: `https://github.com/lovincyrus/personal-vault/releases`
   - Tell the user to check CI for the goreleaser build
