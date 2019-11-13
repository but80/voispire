# go-dio

[![Go Report Card](https://goreportcard.com/badge/github.com/but80/go-dio)](https://goreportcard.com/report/github.com/but80/go-dio)
[![Godoc](https://godoc.org/github.com/but80/go-dio?status.svg)](https://godoc.org/github.com/but80/go-dio)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)

**go-dio** is an unofficial pure Go implementation of **DIO** the fundamental frequency estimation method.

DIO is one feature of [World](https://github.com/mmorise/World) the speech analysis, manipulation and synthesis system.

This version omits the downsampling function. If you want high speed, downsample the input in advance.

## Test

Before testing, you must make these preparations.

- `./tools/make-test` must be compiled by running [./tools/build.sh](./tools/build.sh)
- Some `*.wav` (16 bit, Mono) files must be placed in `./testdata/`.
