#!/bin/sh

# fail if some commands fails
set -e
# show commands
set -x

export CI="openshift"
#make configure-installer-tests-cluster
make bin
mkdir -p $GOPATH/bin
make goget-ginkgo
export PATH="$PATH:$(pwd):$GOPATH/bin"
export CUSTOM_HOMEDIR=$ARTIFACT_DIR

# Copy kubeconfig to temporary kubeconfig file
# Read and Write permission to temporary kubeconfig file
TMP_DIR=$(mktemp -d)
#cp $KUBECONFIG $TMP_DIR/kubeconfig
#chmod 640 $TMP_DIR/kubeconfig
#export KUBECONFIG=$TMP_DIR/kubeconfig

# Login as developer
#oc login -u developer -p password@123
# Login to IBM Cloud using service account API Key
ibmcloud login --apikey $IBMC_OCP47_APIKEY -a cloud.ibm.com -r eu-de -g "Developer CI and QE"

# Login to cluster in IBM Cloud using cluster API key
oc login --token=$IBMC_OCLOGIN_APIKEY --server=$IBMC_OCP47_SERVER

# Check login user name for debugging purpose
oc whoami

# Integration tests
make test-integration
make test-integration-devfile
make test-cmd-login-logout
make test-cmd-project
make test-operator-hub

# E2e tests
make test-e2e-all

cp -r reports $ARTIFACT_DIR 

oc logout
ibmcloud logout
