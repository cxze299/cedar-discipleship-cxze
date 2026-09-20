# Legacy Group Compatibility Implementation Plan

> **For agentic workers:** Implement the following tasks in order; content parsing and migration normalization can proceed independently after agreeing on the data contract.

**Goal:** Preserve the four audited legacy applications' content and completion behavior in Cedar without group-name branches or overwriting current plans.

**Architecture:** Normalize legacy configuration at import. Use explicit `daily.checkin_mode` (`combined` / `separate`) and `weekly_checkin` tasks for aggregate completion. Existing `weekly_book` tasks remain content bindings in aggregate weeks but do not imply individual completion. Reader functions consume normalized schedules and explicit sources.

**Tech Stack:** Go, MySQL, Vue, TypeScript, Vitest.

---

### Task 1: Configuration contract and migration

Files: `backend/cmd/migrate-json/main.go`, `backend/cmd/migrate-json/compatibility.go`, `backend/cmd/migrate-resource-files/main.go`.

- [ ] Normalize top-level `daily_reading` into `task_sections.daily.scripture` with `start_date`, `start_chapter`, `book`, `book_id`, `chapters_per_day`, `books` (objects containing `book`, `book_id`, `chapters`).
- [ ] Preserve `class_rep_shares`, `weekly_reading_catalog`, Markdown paths, and schedule history. Canonical daily type `checkin` means an enabled independent checkin with no invented content plan.
- [ ] Express legacy behavior using `daily.checkin_mode` and `weekly.checkin_mode`; explicit import flags cover source variants with no identifying configuration metadata.
- [ ] Generate aggregate `weekly_checkin` tasks when requested, retaining content bindings. Do not generate empty video/verse placeholders.
- [ ] Parse Chinese record keys and completed status, ignoring revoked entries. Resolve task IDs by exact type/title within the record's group/week; report ambiguous entries.
- [ ] Discover files from configured mounts/references, enforce root containment and existing SHA-256/size/import authorization. Preserve external links.
- [ ] Run `go test ./cmd/migrate-json ./cmd/migrate-resource-files`; exercise all four sanitized configs and verify expected task counts and link sources.

Contract examples:

```json
{"daily":{"checkin_mode":"separate","scripture":{"enabled":true,"type":"checkin"}}}
```

```json
{"weekly":{"checkin_mode":"aggregate","checkin_enabled":true,"reading_path":"/weekly_task.md"}}
```

### Task 2: Reader and frontend

Files: `frontend/src/runtime/content.ts`, `frontend/src/runtime/dailySchedule.ts`, `frontend/src/legacy-app.js`, `frontend/src/components/AppRoot.vue`.

- [ ] Date extraction accepts Chinese dates followed by titles and stops at the next date; date mode takes precedence over numeric headings.
- [ ] Chapter scheduling consumes multiple chapters, crosses books, respects schedule history, clamps pre-start dates as the source does, and ends without wrapping.
- [ ] Open direct HTML links, Markdown weekly sections and reference-only verses; only use verified matching sources. Keep safe Markdown escaping.
- [ ] Render aggregate weekly completion with its content links and independent daily tasks from the authoritative today response.
- [ ] Keep global shares independent from weekly media flags and preserve audio/video/document types.
- [ ] Preserve `weekly_checkin`, week title and daily mode across editor load/save; expose relevant controls without redesign.
- [ ] Verify with Vitest, typecheck and production build.

Expected executable assertions:

```ts
expect(extractNumberedContentSection('### 九月二十日 信心\n正文\n### 九月二十一日 次日\n后文', 0, '九月二十日')).toEqual(['### 九月二十日 信心', '正文']);
```

Full legacy sequence expectations: 2026-05-11 撒下24/王上1; 2026-05-22 王上22/王下1; 2026-09-20 诗77/78; 2027-08-03 启22; 2027-08-04 empty.

### Task 3: Completion model and consumers

Files: `backend/internal/learning/{dto,service}.go`, `backend/internal/checkin/{service,mysql_repository}.go`, `backend/internal/server/checkins.go`, `backend/internal/statistics/`, `backend/internal/notification/`.

- [ ] Add `daily_scripture` and `weekly_checkin` task types; no schema enum change is needed.
- [ ] Derive `WeekVO.weekly_checkin` from its task set and accept the same flag in `WeekInput`. Preserve aggregate week titles.
- [ ] Build separate daily tasks only for enabled components in separate mode; keep current combined mode as default.
- [ ] Validate weekly targets with the existing group/week/type/date checks and locks. Aggregate completion never satisfies a book task.
- [ ] Use the same task identities in today, dashboard task completions, monthly ranking, calendar and notifications.
- [ ] Cover independent daily records, aggregate identity, current-week matching, disabled components and unchanged normal book/video behavior.
- [ ] Run full Go tests and vet, then race checks for learning/checkin/statistics/server.

### Task 4: Existing data and release

Files: `docs/legacy-group-compatibility.md`; narrowly scoped repair tooling if needed.

- [ ] Produce a dry-run diff and database backup for groups 3/4 only. Restore missing fields/bindings with old-value guards; retain all original data in the audit backup.
- [ ] Do not force-import groups, create absent groups, overwrite `agape-a`, or copy another group's missing source.
- [ ] Validate real full-year devotion files and every day in the complete scripture sequence.
- [ ] Commit with description, fast-forward master preserving unrelated working changes, push, build locally for Linux amd64 and upload the small runtime context.
- [ ] Deploy affected services; verify health, asset availability, frontend hash and authenticated today/dashboard behavior.
- [ ] Document confirmed compatibility and missing legacy source files.
