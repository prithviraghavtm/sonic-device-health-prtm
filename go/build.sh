#! /bin/bash

# may have to set GO111MODULE=off

# export GOPATH=$(pwd;)

# echo $GOPATH

# set -x

install() {
    echo "run install"
    go install ./...
}

test() {
  go test $1 -coverprofile=coverprofile.out  -coverpkg lib/lomipc,lib/lomcommon -covermode=atomic lib/lib_test
  if [ $? -ne 0 ]; then
        echo "Failed to run test"
        exit -1
  fi

  go tool cover -html=coverprofile.out -o /tmp/coverage.html
  ls -l coverprofile.out /tmp/coverage.html
  echo "View /tmp/coverage.html in Edge"
}

etest() { 
  go test $1 -p 1 -coverprofile=dbclient_coverprofile.out  -coverpkg go/src/vendors/sonic/client/dbclient -covermode=atomic go/src/vendors/sonic/client/dbclient
  if [ $? -ne 0 ]; then
  	echo "Failed to run test"
  	exit -1
  fi
  cat ./dbclient_coverprofile.out* > dbclient_coverprofile.out
  go tool cover -html=dbclient_coverprofile.out -o /tmp/dbclient_coverage.html
  ls -l dbclient_coverprofile.out /tmp/dbclient_coverage.html
  echo "View /tmp/dbclient_coverage.html in Edge"

}

clean() {
    echo "run clean"
    rm -rf bin/*
    rm -rf pkg/*
    rm -f coverprofile.out
}

list() {
    tree0
}

cmd="None"

if [ $# -ne 0 ]; then
    cmd=$1
fi  

if [[ "install" == "$cmd"* ]]; then
    install

elif [[ "clean" == "$cmd"* ]]; then
    clean

elif [[ "list" == "$cmd"* ]]; then
    list

elif [[ "test" == "$cmd"* ]]; then
    test ""

elif [[ "etest" == "$cmd"* ]]; then
    etest ""

elif [[ "vtest" == "$cmd"* ]]; then
    test "-v"

else
    echo "($cmd) match None"
    echo "install / test / clean"
fi

