#!/bin/bash
set -x
# Setup to find nessasary data from cluster setup
## Constants
LIBDIR="./scripts/configure-cluster"
LIBCOMMON="$LIBDIR/common"
SETUP_OPERATORS="$LIBCOMMON/setup-operators.sh"
SETUP_POSTGRES_OPERATOR="$LIBCOMMON/setup-postgres-operator.sh"
AUTH_SCRIPT="$LIBCOMMON/auth.sh"
KUBEADMIN_SCRIPT="$LIBCOMMON/kubeconfigandadmin.sh"

CI_OPERATOR_HUB_PROJECT="ci-operator-hub-project"
POSTGRES_OPERATOR_PROJECT="odo-operator-test"

# list of namespace to create
IMAGE_TEST_NAMESPACES="openjdk-11-rhel8 nodejs-12-rhel7 nodejs-12 openjdk-11 nodejs-14"

. $KUBEADMIN_SCRIPT

. $SETUP_POSTGRES_OPERATOR

# Setup the cluster for Operator tests

# Create a new namesapce which will be used for OperatorHub checks
oc new-project $CI_OPERATOR_HUB_PROJECT
# Let developer user have access to the project
oc adm policy add-role-to-user edit developer

sh $SETUP_OPERATORS

oc new-project $POSTGRES_OPERATOR_PROJECT
# Let developer user have access to the project
oc adm policy add-role-to-user edit developer

install_postgres_operator $POSTGRES_OPERATOR_PROJECT

# OperatorHub setup complete

# Create the namespace for e2e image test apply pull secret to the namespace
for i in `echo $IMAGE_TEST_NAMESPACES`; do
    # create the namespace
    oc new-project $i
    # Applying pull secret to the namespace which will be used for pulling images from authenticated registry
    oc get secret pull-secret -n openshift-config -o yaml | sed "s/openshift-config/$i/g" | oc apply -f -
    # Let developer user have access to the project
    oc adm policy add-role-to-user edit developer
done

# Workarounds - Note we should find better soulutions asap
# Missing wildfly in OpenShift Adding it manually to cluster Please remove once wildfly is again visible
oc apply -n openshift -f https://raw.githubusercontent.com/openshift/library/master/arch/x86_64/community/wildfly/imagestreams/wildfly-centos7.json

sh $AUTH_SCRIPT
