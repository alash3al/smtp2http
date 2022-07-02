# Sender Policy Framework

A comprehensive RFC7208 implementation

[![Build Status](https://travis-ci.org/zaccone/spf.svg?branch=master)](https://travis-ci.org/zaccone/spf)
[![Go Report Card](https://goreportcard.com/badge/github.com/zaccone/spf)](https://goreportcard.com/report/github.com/zaccone/spf)
[![GoDoc](https://godoc.org/github.com/zaccone/spf?status.svg)](https://godoc.org/github.com/zaccone/spf)

## About
The SPF Library implements Sender Policy Framework described in RFC 7208. It aims to cover all rough edge cases from RFC 7208.
Hence, the library does not operate on strings only, rather "understands" SPF records and reacts properly to valid and invalid 
input. Wherever I found it useful, I added comments with RFC sections and quotes directly in the source code, so the readers can follow 
implemented logic.

## Current status
The library is still under development. API may change, including function/methods names and signatures. I will consider it correct and stable once it passess all tests described in the most popular SPF implementation - pyspf.

## Testing
Testing is an important part of this implementation. There are unit tests that will run locally in your environment, however there are 
also configuration files for `named` DNS server that would be able to respond  implemented testcases. (In fact, for the long time I used a 
real DNS server with such configuration as a testing infrastructure for my code).
There is a plan to implement simple DNS server that would be able to read .yaml files with comprehensive testsuite defined in pyspf package. Code coverage is also important part of the development and the aim is to keep it as high as 9x %

## Dependencies
SPF library depends on another [DNS](https://github.com/miekg/dns) library. Sadly, Go's builtin DNS library is not elastic enough and does not allow for controlling 
underlying DNS queries/responses.

## Pull requests & code review
If you have any comments about code structure feel free to reach out or simply make a Pull Request

