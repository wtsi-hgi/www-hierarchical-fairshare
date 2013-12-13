#!/bin/bash

export CGO_LDFLAGS="-L/usr/local/lsf/9.1/linux2.6-glibc2.3-x86_64/lib"
export CGO_CFLAGS="-I/usr/local/lsf/9.1/include"

go build hfs_httpserver.go
