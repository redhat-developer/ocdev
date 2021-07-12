#!/bin/bash
set -x

install_redis_operator(){
  $1 create -f - <<EOF
  apiVersion: operators.coreos.com/v1alpha1
  kind: Subscription
  metadata:
    name: my-redis-operator
    namespace: $2
  spec:
    channel: stable
    name: redis-operator
    source: $3
    sourceNamespace: $4
EOF
}

install_service_binding_operator(){
  $1 create -f - <<EOF
  apiVersion: operators.coreos.com/v1alpha1
  kind: Subscription
  metadata:
    name: my-service-binding-operator
    namespace: $2
  spec:
    channel: beta
    name: $3
    source: $4
    sourceNamespace: $5
EOF
}

if [ $KUBERNETES == "true" ]; then
  # install "redis-oprator" using "kubectl" in "operators" namespace; use "operatorhubio-catalog" catalog source from "olm" namespace
  install_redis_operator kubectl operators operatorhubio-catalog olm
else
  # install "redis-oprator" using "oc" in "openshift-operators" namespace; use "community-operators" catalog source from "openshift-marketplace" namespace
  install_redis_operator oc openshift-operators community-operators openshift-marketplace

  # install "service-binding-operator" using "oc" in "openshift-operators" namespace; use "redhat-operators" catalog source from "openshift-marketplace" namespace
  install_service_binding_operator oc openshift-operators rh-service-binding-operator redhat-operators openshift-marketplace
fi