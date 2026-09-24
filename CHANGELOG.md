# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Fixed

- Restored signed streaming for MP3 and other audio assets used by weekly media tasks. This closes the frontend/backend contract gap that caused `asset_not_video` after the streaming frontend was deployed without its matching backend change.

### Operations

- Require every non-merge commit to update this changelog.
