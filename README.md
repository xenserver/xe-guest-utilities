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

With modern go versions (later than 1.11)
-----------
Type `make` or `make build` to build the xenstore and xe-daemon.

* The binarys will be in `build/obj`
* In `build/stage` are all required files and where they go when installed.
* In `build/dist` is a tarball with all files,symlinks and permissions.


Older Go versions
-----------

Earliest version that has all required features is `1.10`

1. `GOPATH` 

Go gets librarys from the `GOPATH`, so for this to work, you need read/write permissions there.
If in doubt, set `GOPATH` to a temporary location, ie: `export GOPATH=$(pwd)` sets `GOPATH` to the local folder

2. external library

This project uses the `golang.org/x/sys/unix` library, so you will need to install that:

`go get golang.org/x/sys/unix`

this will install it and all its dependency's to `$GOPATH/src`.

3. Get this project

For go to find all files in this project it needs to be in the `GOPATH`
This is easiest done by just putting it into `$GOPATH/src/xe-guest-utilities`

`git clone https://github.com/xenserver/xe-guest-utilities.git $GOPATH/src/xe-guest-utilities` 

4 Build

Go into the right directory `cd $GOPATH/src/xe-guest-utilities/`

now you can `make build` or `make`.

resulting files are in `build/`, same layout as explained above

