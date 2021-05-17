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

USE $GOPATH to build
-----------
e.g. use go1.6.4 to build
1. Set GOPATH

`export GOPATH=/root/go`

2. Download xe-guest-utilities

`mkdir -p $GOPATH/src; cd $GOPATH/src`

`git clone git@github.com:xenserver/xe-guest-utilities.git`
3. Disable GO111MODULE

`export GO111MODULE=off`

4. Download golang.org/x/sys

`mkdir -p $GOPATH/src/golang.org/x`

`cd $GOPATH/src/golang.org/x`

`git clone http://github.com/golang/sys.git`

5. Build

`cd $GOPATH/src/xe-guest-utilities`

`make`

Enable $GO111MODULE to build
-----------
e.g. use go1.6.4 to build
1. Download xe-guest-utilities

`cd /root`

`git clone git@github.com:xenserver/xe-guest-utilities.git`
2. Enable GO111MODULE

`export GO111MODULE=on`

3. Initializing a project

`go mod init xe-guest-utilities`

`go mod tidy`

4. Build

`cd /root/xe-guest-utilities.git`

`make`
