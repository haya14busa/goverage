## goverage - go test -coverprofile for multiple packages

[![CircleCI](https://circleci.com/gh/haya14busa/goverage.svg?style=svg)](https://circleci.com/gh/haya14busa/goverage)
[![LICENSE](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

The solution of https://github.com/golang/go/issues/6909 with one binary.

## Installation

```
go get github.com/haya14busa/goverage
```

## Usage

```
Usage:  goverage [flags] -coverprofile=coverage.out package

Flags:
  -covermode string
        sent as covermode argument to go test
  -coverprofile string
        Write a coverage profile to the file after all tests have passed. (required)
  -cpu string
        sent as cpu argument to go test
  -parallel string
        sent as parallel argument to go test
  -short
        sent as short argument to go test
  -timeout string
        sent as timeout argument to go test
  -v    sent as v argument to go test
```

### :bird: Author
haya14busa (https://github.com/haya14busa)
