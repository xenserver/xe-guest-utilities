[![Build Status](https://travis-ci.org/xenserver/xe-guest-utilities.svg?branch=master)](https://travis-ci.org/xenserver/xe-guest-utilities)

Introduction
===================

This is the golang guest utilities for XenServer


XenStore CLI
-----------
xe-guest-utilities.git/xenstore


XenServer Guest Utilities
-----------
xe-guest-utilities.git/xe-daemon


Build Instructions
===================
[Go development environment](https://golang.org/doc/install) is required to build the guest utilities.

Type `make` or `make build` to build the xenstore and xe-daemon.

Build with GO111MODULE disabled
-----------
1. Set GOPATH

`export GOPATH=/root/go`

2. Download xe-guest-utilities

`mkdir -p $GOPATH/src; cd $GOPATH/src`

`git clone git@github.com:xenserver/xe-guest-utilities.git`

3. Disable GO111MODULE

`export GO111MODULE=off`

4. Download golang.org/x/sys

`go get golang.org/x/sys`

5. Build

`cd $GOPATH/src/xe-guest-utilities`

`make`

Build with GO111MODULE Enabled
-----------
1. Download xe-guest-utilities

`cd /root`

`git clone git@github.com:xenserver/xe-guest-utilities.git`
2. Enable GO111MODULE

Check if GO111MODULE is enabled, if not execute `export GO111MODULE=on`

3. Build

`cd /root/xe-guest-utilities.git`

`make`
