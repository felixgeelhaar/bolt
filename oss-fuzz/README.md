# OSS-Fuzz integration files

This directory holds the three files OSS-Fuzz expects under
`projects/bolt/` in the
[google/oss-fuzz](https://github.com/google/oss-fuzz) repository:

- `project.yaml` — project metadata (homepage, language, contact)
- `Dockerfile` — build environment
- `build.sh` — compiles bolt's `Fuzz*` targets into native fuzzers

## How to onboard

This isn't auto-picked up. The maintainer needs to:

1. Apply for a project at
   [oss-fuzz onboarding](https://google.github.io/oss-fuzz/getting-started/new-project-guide/).
2. Open a PR against `google/oss-fuzz` adding `projects/bolt/` with
   the three files in this directory.
3. Wait for OSS-Fuzz reviewers to merge; continuous fuzzing starts
   automatically after merge.
4. Subscribe to the OSS-Fuzz issue tracker for crash reports —
   they'll be filed against the listed contact addresses.

The files here are kept in this repo so they version with the fuzz
targets they reference. When `fuzz_test.go` gains or loses targets,
update `build.sh` here in the same PR so the OSS-Fuzz copy can be
synced.

## Targets currently exposed

`build.sh` compiles each of these:

- `FuzzJSONHandler`
- `FuzzInputValidation`
- `FuzzFloatFormatting`
- `FuzzBufferManagement`
- `FuzzConcurrentLogging`
- `FuzzKeyValidation`
- `FuzzUnicodeHandling`
- `FuzzMessageEscaping`
- `FuzzLevelValidation`
