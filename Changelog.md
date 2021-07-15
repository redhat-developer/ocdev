#  Odo Changelog

## Next

### Feature/Enhancements

### Bug Fixes

### Tests

### Documentation

## 2.2.x

### Feature/Enhancements

### Bug Fixes
- Fix component create and delete help examples([#4904](https://github.com/openshift/odo/pull/4904))

### Tests

### Documentation

## 2.2.3

### Feature/Enhancements

- `odo service delete` removes Operator backed services from the Devfile only, without undeploying them from the cluster ([#4761](https://github.com/openshift/odo/pull/4761))
- `odo service list` lists Operator backed services in Devfile and/or deployed into the cluster and their state (Pushed, Not pushed, Deleted locally) ([#4761](https://github.com/openshift/odo/pull/4761))
- `odo push` undeploys Operator backed services marked as managed by the current devfile not present in this devfile anymore ([#4761](https://github.com/openshift/odo/pull/4761))
- param based `odo service create` for operator backed services ([#4704](https://github.com/openshift/odo/pull/4704))
- add `odo catalog describe service <operator> --example` ([#4821](https://github.com/openshift/odo/pull/4821))
- `odo link` and `odo unlink` write to devfile without deploying to cluster. Deploying happens when running `odo push` ([#4819](https://github.com/openshift/odo/pull/4819))
- `odo describe` shows linked services and components whether they are pushed or not. When deployed, it also shows environment variables or mounted files ([#4866](https://github.com/openshift/odo/pull/4866))
- collect component type and cluster type data for telemetry ([#4763](https://github.com/openshift/odo/pull/4763))

### Bug Fixes
- Update telemetry contextual properties only when it is enabled([#4870](https://github.com/openshift/odo/pull/4870))

- Completely move to using networking v1 and extensions v1 ingresses based on cluster support as extensions v1 ingress is deprecated ([#4853](https://github.com/openshift/odo/pull/4853))

## 2.2.2

### Feature/Enhancements

- Custom link name and bind-as-files option for `odo link` ([#4729](https://github.com/openshift/odo/pull/4729))
- `odo list` now lists components created/managed by other tools ([#4742](https://github.com/openshift/odo/pull/4742))

### Bug Fixes

- Adding KUBECONFIG checks in odo catalog list components ([#4756](https://github.com/openshift/odo/pull/4756))
- use filepath Join instead of / while constructing kubeconfig path ([#4765](https://github.com/openshift/odo/pull/4765))
- `odo push` can deploy new services when a service is already deployed ([#4772](https://github.com/openshift/odo/pull/4772))
- `odo env set DebugPort` doesn't work for converted devfile ([#4785](https://github.com/openshift/odo/pull/4785))
- Sanitize Telemetry data ([#4758](https://github.com/openshift/odo/pull/4758))

### Tests

- Refactor devfile delete tests and add validity checks for delete command ([#4793](https://github.com/openshift/odo/pull/4793))

### Documentation

- Enhance usage data documentation ([#4774](https://github.com/openshift/odo/pull/4774))

## 2.2.1

### Feature/Enhancements

- Implement `odo catalog describe service` for operator backed services ([#4671](https://github.com/openshift/odo/pull/4671))
- Add deprecation warning for old git style devfile registries ([#4707](https://github.com/openshift/odo/pull/4707))
- Adds dev.odo.push.path attribute support for pushing only mentioned files ([#4588](https://github.com/openshift/odo/pull/4588))
- Use server side apply  approved ([#4648](https://github.com/openshift/odo/pull/4648))
- Adding wait support to component deletion for devfile ([#4712](https://github.com/openshift/odo/pull/4712))
- Collect Component type for usage data ([#4662](https://github.com/openshift/odo/pull/4662))

### Bug Fixes

- Follow devfile like conventions in generated url name to keep url short for --s2i  ([#4670](https://github.com/openshift/odo/pull/4670))
- Fix OCI-based registry migration  approved kind/bug lgtm ([#4702](https://github.com/openshift/odo/pull/4702))
- Removes invalid endpoints from the devfile on triggering url create. ([#4567](https://github.com/openshift/odo/pull/4567))

### Tests

- Automate psi ci for mac and windows  ([#4460](https://github.com/openshift/odo/pull/4460))
- Update devfile tests with OCI-based registry ([#4679](https://github.com/openshift/odo/pull/4679))


### Documentation

- Adds a document regarding the odo.dev.push.path attributes in the devfile ([#4681](https://github.com/openshift/odo/pull/4681))
- Add --s2i conversion related breaking changes ([#4683](https://github.com/openshift/odo/pull/4683))
- Fix OCI-based registry migration ([#4702](https://github.com/openshift/odo/pull/4702))
