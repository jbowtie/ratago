ratago
======

[![Build Status](https://travis-ci.org/jbowtie/ratago.svg?branch=master)](https://travis-ci.org/jbowtie/ratago)
[![codecov](https://codecov.io/gh/jbowtie/ratago/branch/master/graph/badge.svg)](https://codecov.io/gh/jbowtie/ratago)
[![Go Report Card](https://goreportcard.com/badge/github.com/jbowtie/ratago)](https://goreportcard.com/report/github.com/jbowtie/ratago)
[![GoDoc](https://godoc.org/github.com/jbowtie/ratago?status.svg)](https://godoc.org/github.com/jbowtie/ratago)

Ratago is a (mostly-compliant) implementation of an XSLT 1.0 processor written in Go and released under an MIT license.

Currently it should be seen as experimental - it lacks full compliance with the spec. It has been run successfully on a number of scripts of moderate complexity as of the 0.4-pre release.

The test suite is derived from the test suite used by the libxslt library written by Daniel Veillard. See http://xmlsoft.org/XSLT/ for details on libxslt.

Installation
----

For MacOS:
```sh
# Need pkg-config, see https://stackoverflow.com/a/36794452/700471
brew install pkg-config 
# Need libxml2 source, see https://github.com/mitmproxy/mitmproxy/issues/68#issuecomment-120301708
brew install libxml2
sudo ln -s /usr/local/opt/libxml2/include/libxml2/libxml /usr/local/include/libxml 
# Install with Go Modules
GO111MODULE=on go get github.com/jbowtie/ratago
```

TODO
----

There are several tasks remaining to reach full compliance. Until these tasks are complete the API is subject to change.

* Implement xsl:decimal-format and format-number.
* Ensure that errors are properly progogated in Go fashion.

