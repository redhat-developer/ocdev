#  Odo Changelog

## 2.2.2

### Feature/Enhancements

- Custom link name and bind-as-files option for `odo link` ([#4729](https://github.com/openshift/odo/pull/4729))

### Bug Fixes

- Adding KUBECONFIG checks in odo catalog list components ([#4756](https://github.com/openshift/odo/pull/4756))
- use filepath Join instead of / while constructing kubeconfig path ([#4765](https://github.com/openshift/odo/pull/4765))
- `odo push` can deploy new services when a service is already deployed ([#4772](https://github.com/openshift/odo/pull/4772))
- odo env set DebugPort doesn't work for converted devfile([#4785](https://github.com/openshift/odo/pull/4785))

### Tests

### Documentation

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
