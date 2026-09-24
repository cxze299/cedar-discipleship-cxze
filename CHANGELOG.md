# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Changed

- Reduce first-load JavaScript by registering only used UI components and loading management, statistics, ministry, and reading pages on demand. Compress static scripts and styles on NAS and reuse the installed PingFang Semibold font when available, while preserving signed audio/video streaming and Range requests.
- Put the management workspace directly in mobile navigation for authorized users, with navigation width adapting to the available entries.
- Consolidate management actions in `AdminConsole` and share page-loading/error handling. Preserve the deployed NAS frontend features, including recitation practice, download handling, and existing mobile layouts.

### Fixed

- Discard stale management, ministry-workspace, and content-viewer responses after navigation or group changes. Surface management load failures and allow retries instead of caching an empty resource library.
- Restored signed streaming for MP3 and other audio assets used by weekly media tasks. This closes the frontend/backend contract gap that caused `asset_not_video` after the streaming frontend was deployed without its matching backend change.

### Operations

- Require every non-merge commit to update this changelog.
