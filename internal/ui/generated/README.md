This directory is the stable embed root for generated harness UI assets.

Tracked files in this directory exist only to keep the embed path present in a
clean checkout and to document the contract. The actual production UI bundle is
generated under `internal/ui/generated/build/` by `scripts/build-embedded-ui`,
`scripts/install-dev-harness`, CI, and release automation.
