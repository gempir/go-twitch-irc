# Benchmark

## Setup git lfs

Checkout https://git-lfs.github.com/ on how to setup git lfs for your system

Run `git lfs install --local` to setup the hooks for the repo

Run `git lfs pull` to download all lfs objects for the current ref

## Run benchmarks

To only run the benchmarks do `go test -bench=. -run=^a`

The -run flag with those parameters prevents any test vom running since none of them can start with a.

The CI runs like this `go test -bench=. -benchtime=5x -run=^a` to make sure enough multiple passes are done

or just run `make bench`
