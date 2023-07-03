#!/usr/bin/env bash

##########
VERSION=${1}

# These are the values we want to pass for Version and BuildTime
GITHASH=`git rev-parse HEAD 2>/dev/null`

BUILDAT=`date +%FT%T%z`

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS="-s -w -X github.com/TruthHun/BookStack/utils.GitHash=${GITHASH} -X github.com/TruthHun/BookStack/utils.BuildAt=${BUILDAT} -X github.com/TruthHun/BookStack/utils.Version=${VERSION}"

##########

rm -rf output/${VERSION}
mkdir -p output/${VERSION}

CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -v -o output/${VERSION}/mac/BookStack -ldflags "${LDFLAGS}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o output/${VERSION}/linux/BookStack -ldflags "${LDFLAGS}"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -v -o output/${VERSION}/windows/BookStack.exe -ldflags "${LDFLAGS}"

upx -f -9 output/${VERSION}/mac/BookStack
upx -f -9 output/${VERSION}/linux/BookStack
upx -f -9 output/${VERSION}/windows/BookStack.exe

cp -r conf output/${VERSION}/mac/
cp -r conf output/${VERSION}/linux/
cp -r conf output/${VERSION}/windows/

cp -r views output/${VERSION}/mac/
cp -r views output/${VERSION}/linux/
cp -r views output/${VERSION}/windows/

cp -r dictionary output/${VERSION}/mac/
cp -r dictionary output/${VERSION}/linux/
cp -r dictionary output/${VERSION}/windows/

cp -r static output/${VERSION}/mac/
cp -r static output/${VERSION}/linux/
cp -r static output/${VERSION}/windows/