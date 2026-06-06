#!/bin/bash -eu
#
# OSS-Fuzz build script. Compiles each FuzzXxx target in
# fuzz_test.go to a native libFuzzer-compatible binary placed in
# $OUT/. Each target gets its own corpus directory if one exists
# under testdata/fuzz/.
#
# Sync with fuzz_test.go: when targets are added or removed there,
# update the TARGETS list below in the same PR.

cd $SRC/bolt

TARGETS=(
  FuzzJSONHandler
  FuzzInputValidation
  FuzzFloatFormatting
  FuzzBufferManagement
  FuzzConcurrentLogging
  FuzzKeyValidation
  FuzzUnicodeHandling
  FuzzMessageEscaping
  FuzzLevelValidation
)

for target in "${TARGETS[@]}"; do
  compile_native_go_fuzzer go.klarlabs.de/bolt "$target" "$target"

  # Seed corpus, if present.
  corpus_dir="testdata/fuzz/${target}"
  if [ -d "${corpus_dir}" ]; then
    zip -j "$OUT/${target}_seed_corpus.zip" "${corpus_dir}"/*
  fi
done
