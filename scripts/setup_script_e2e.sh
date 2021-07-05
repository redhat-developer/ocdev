#!/usr/bin/env bash

shout() {
  set +x
  echo -e "\n!!!!!!!!!!!!!!!!!!!!\n${1}\n!!!!!!!!!!!!!!!!!!!!\n"
  set -x
}

set -ex

shout "Setting up some stuff"

# Create a bin directory whereever script runs. This will be where all binaries that need to be in PATH will reside.
mkdir bin artifacts
# Change the default location of go's bin directory (without affecting GOPATH). This is where compiled binaries will end up by default
# for eg go get ginkgo later on will produce ginkgo binary in GOBIN
export GOBIN="`pwd`/bin"
export GOBIN_TEMP=$GOBIN
# Set kubeconfig to current dir. This ensures no clashes with other test runs
export KUBECONFIG="`pwd`/config"
export ARTIFACT_DIR=${ARTIFACT_DIR:-"`pwd`/artifacts"}
export CUSTOM_HOMEDIR=$ARTIFACT_DIR
export WORKDIR=${WORKDIR:-"`pwd`"}
export GOCACHE=`pwd`/.gocache && mkdir $GOCACHE

# This si one of the variables injected by ci-firewall. Its purpose is to allow scripts to handle uniqueness as needed
SCRIPT_IDENTITY=${SCRIPT_IDENTITY:-"def-id"}
export SKIP_USER_LOGIN_TESTS="true"

shout "Getting oc binary"
if [[ $BASE_OS == "linux"  ]]; then
    set +x
    curl --connect-timeout 150 --max-time 2048 -k ${OCP4X_DOWNLOAD_URL}/${ARCH}/${BASE_OS}/oc.tar -o ./oc.tar
    set -x
    tar -C $GOBIN -xvf ./oc.tar && rm -rf ./oc.tar
else
    set +x
    curl --connect-timeout 210 --max-time 2048 -k ${OCP4X_DOWNLOAD_URL}/${ARCH}/${BASE_OS}/oc.zip -o ./oc.zip
    set -x
    if [[ $BASE_OS == "windows" ]]; then
        GOBIN_TEMP=$GOBIN
        GOBIN="$(cygpath -pw $GOBIN)"
        CURRDIR="$(cygpath -pw $WORKDIR)"
        GOCACHE="$(cygpath -pw $GOCACHE)"
        powershell -Command "Expand-Archive -Path $CURRDIR\oc.zip  -DestinationPath $GOBIN"
        chmod +x $GOBIN_TEMP/*
    fi
    if [[ $BASE_OS == "mac" ]]; then
        # Skiping download of oc binary for test due to bandwidth issue, binary from cluster is stored in ~/ocforodo dir in macmini 
        #unzip ./oc.zip -d ~/go/bin/ && chmod +x ~/go/bin/oc
        unzip ./oc.zip -d $GOBIN && rm -rf ./oc.zip && chmod +x $GOBIN/oc
        PATH="$PATH:/usr/local/bin:/usr/local/go/bin"
    fi
fi

# Add GOBIN which is the bin dir we created earlier to PATH so any binaries there are automatically available in PATH
export PATH=$PATH:$GOBIN

#-----------------------------------------------------------------------------