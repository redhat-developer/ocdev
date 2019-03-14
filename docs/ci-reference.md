# How to run integration test job in Travis CI

For default oc, use the configuration in .travis.yaml. For example:

```sh
  # Run main e2e tests
    - <<: *base-test
      stage: test
      name: "Main e2e tests"
      script:
        - ./scripts/oc-cluster.sh
        - make bin
        - sudo cp odo /usr/bin
        - oc login -u developer
        - make test-main-e2e
```

If the need presents itself to run odo integration tests against a specific version of Openshift, use env variable `OPENSHIFT_CLIENT_BINARY_URL` to pass the [released](https://github.com/openshift/origin/releases) oc client URL in `.travis.yaml`. For oc v3.10.0, use the configuration:

```sh
  # Run main e2e tests
    - <<: *base-test
      stage: test
      name: "Main e2e tests"
      script:
        - OPENSHIFT_CLIENT_BINARY_URL=https://github.com/openshift/origin/releases/download/v3.10.0/openshift-origin-client-tools-v3.10.0-dd10d17-linux-64bit.tar.gz ./scripts/oc-cluster.sh
        - make bin
        - sudo cp odo /usr/bin
        - oc login -u developer
        - make test-main-e2e
```

# How to run integtration tests on Prow

Prow is the Kubernetes / OpenShift way of managing workflow including tests. To get tests on there, you need to raise PR to openshift/release repository setting up appropriate ci operator config and job files. Reference for same is available there itself.

However note that prow gives you a bare bones cluster, so we need to preconfigure the same so that the cluster is in state expected such as auth being configured and so on.

You can do this by running `$ make configure-installer-tests-cluster` before running actual tests.
This script is configurable with environment variables as

 - CI: If this environment is set, then initial setup is skipped in favor of only configuring authentication. Use only with OpenShift CI
 - DEFAULT_INSTALLER_ASSETS_DIR: The location where OpenShift installer creates assets such as kube admin password and the kubeconfig file.
