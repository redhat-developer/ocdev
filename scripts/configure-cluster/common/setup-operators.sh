#!/bin/bash
set -x

# By default the runner 
export RUNNER="oc"
if [ $KUBERNETES == "true" ]; then
  RUNNER="kubectl"
fi

install_etcd_operator(){
  # Create subscription
  $RUNNER create -f - <<EOF
  apiVersion: operators.coreos.com/v1alpha1
  kind: Subscription
  metadata:
    name: etcd
    namespace: openshift-operators
  spec:
    channel: clusterwide-alpha
    installPlanApproval: Automatic
    name: etcd
    source: community-operators
    sourceNamespace: openshift-marketplace
EOF
}

install_service_binding_operator(){
  $RUNNER create -f - <<EOF
  apiVersion: operators.coreos.com/v1alpha1
  kind: Subscription
  metadata:
    labels:
      operators.coreos.com/rh-service-binding-operator.openshift-operators: ""
    name: rh-service-binding-operator
    namespace: openshift-operators
  spec:
    channel: beta
    installPlanApproval: Automatic
    name: rh-service-binding-operator
    source: redhat-operators
    sourceNamespace: openshift-marketplace
EOF
}

# install etcd operator

install_etcd_operator

# install service-binding-operator

install_service_binding_operator
