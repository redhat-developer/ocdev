#!/bin/bash
set -x

install_etcd_operator(){
  # Create subscription
  oc create -f - <<EOF
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
    startingCSV: etcdoperator.v0.9.4-clusterwide
EOF
}

install_service_binding_operator(){
  oc create -f - <<EOF
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
    startingCSV: service-binding-operator.v0.3.0
EOF
}

# install etcd operator

install_etcd_operator

# install service-binding-operator

install_service_binding_operator
