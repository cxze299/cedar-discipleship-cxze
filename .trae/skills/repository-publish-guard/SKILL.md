---
name: repository-publish-guard
description: Audits staged files for repository relevance and private local content. Use before every commit or push, and when deciding whether a skill, script, report, or configuration belongs in the repository.
---

# Repository Publish Guard

Keep the repository limited to files that collaborators need to build, test, understand, or operate the product in supported environments.

## Required Check

Run this check immediately before every commit and push:

```bash
git status --short
git diff --cached --name-status
git diff --cached --stat
git ls-files -ci --exclude-standard
```

Review every staged path. Do not approve a commit from the summary alone; inspect the staged diff.

## Repository-Worthy Files

Keep a file staged only when all of these are true:

- It directly implements, tests, documents, configures, or supports a repository feature.
- Another contributor or supported deployment environment needs it.
- It contains no personal machine identity, private endpoint, credential, local absolute path, or transient evidence.
- Its behavior and naming are understandable without the author's private environment.

A project Skill belongs in the repository only when it teaches reusable repository-wide behavior needed by other contributors. A Skill for one developer's NAS, workstation, account, local validation flow, or private deployment target stays local.

## Remove From The Commit

Unstage files in any of these categories unless the user explicitly confirms they are shared project assets:

- Local deployment or verification scripts tied to a private machine or service.
- Private Skills containing local hosts, usernames, ports, filesystem paths, tokens, or personal operating procedures.
- `.env` files, credentials, private keys, cookies, tokens, database dumps, and authentication responses.
- Debug sessions, `.dbg/`, runtime logs, screenshots, temporary reports, benchmark output, generated bundles, and build artifacts.
- IDE state, local caches, temporary directories, and machine-specific configuration.
- Notes, plans, or documents unrelated to the product or unnecessary for other repository users.
- Opportunistic files unrelated to the requested change.

Unstage without deleting the local file:

```bash
git restore --staged -- <path>
```

For a file that was already tracked but must become local-only:

```bash
git rm --cached -- <path>
```

Add it to `.git/info/exclude` when the exclusion is specific to this checkout. Add it to `.gitignore` only when every contributor should ignore the same class of file.

## Content Scan

Inspect staged content for likely private data without printing matched values:

```bash
git grep --cached -I -l -E \
  '(/Users/[^/]+|/volume[0-9]+/|BEGIN [A-Z ]*PRIVATE KEY|NAS_PASSWORD|AGP_TEST_PASSWORD|Authorization: Bearer)' \
  -- .
```

Treat matches as review prompts, not automatic failures. Test fixtures and generalized documentation may intentionally contain representative values, but the staged diff must prove they are required and non-sensitive.

Review every intentionally forced ignored file:

```bash
git ls-files -ci --exclude-standard
```

An ignored file may remain tracked only when it is deliberately shared, repository-relevant, and free of local private content.

## Final Gate

Before committing:

1. Confirm each staged file maps directly to the requested change.
2. Confirm `CHANGELOG.md` is included and its newest entry is in Chinese.
3. Confirm no local-only or unrelated file is staged.
4. Run `git diff --cached --check`.
5. Show the final staged file list with `git diff --cached --name-status`.

Before pushing, compare the commit to its upstream base:

```bash
git diff --name-status origin/master...HEAD
git log --format='%h %s' origin/master..HEAD
```

Stop and unstage anything that fails this gate. Never rewrite already-published history merely to remove a non-secret local file without explicit user authorization; remove it from the current tree in a new commit and report that older history still contains it.
