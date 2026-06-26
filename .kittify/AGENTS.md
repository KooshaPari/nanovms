# Agent Rules (mirror of _cu_audit/templates/.kittify/AGENTS.md)

**All AI agents working in this project must follow these rules.**

## 1. Path Reference

When mentioning directories or files, provide either the absolute path or a
path relative to the project root.

## 2. UTF-8 Encoding

When writing ANY markdown, JSON, YAML, CSV, or code files, use ONLY UTF-8
compatible characters. Avoid Windows-1252 smart quotes, em/en dashes, and
copy-paste arrows. Use ASCII equivalents.

## 3. Context Management

Build the context you need, then maintain it intelligently. Don't re-read
files mid-session unless the file has changed.

## 4. Work Quality

Produce secure, tested, documented work. Treat security warnings as fatal.

## 5. Git Discipline

Commit only meaningful units of work. Descriptive commit messages in
imperative mood. No rewriting history of shared branches.

## 6. Agent Directories

NEVER commit agent directories (`.claude/`, `.codex/`, `.gemini/`, etc.) to
git. They contain auth tokens and session data.

### Worktree Constitution Sharing

In worktrees, `.kittify/memory/` is a symlink to the main repo's memory,
ensuring all feature branches share the same constitution.

## Quick Reference

- Path: always specify exact locations
- Encoding: UTF-8 only
- Context: read what you need, don't re-read unnecessarily
- Quality: secure, tested, documented
- Git: clean commits, descriptive messages
