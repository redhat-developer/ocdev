PROJECT := github.com/openshift/odo
ifdef GITCOMMIT
        GITCOMMIT := $(GITCOMMIT)
else
        GITCOMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)
endif

COMMON_GOFLAGS := -mod=vendor
COMMON_LDFLAGS := -X $(PROJECT)/pkg/version.GITCOMMIT=$(GITCOMMIT)
BUILD_FLAGS := $(COMMON_GOFLAGS) -ldflags="$(COMMON_LDFLAGS)"
CROSS_BUILD_FLAGS := $(COMMON_GOFLAGS) -ldflags="-s -w -X $(PROJECT)/pkg/segment.writeKey=R1Z79HadJIrphLoeONZy5uqOjusljSwN $(COMMON_LDFLAGS)"
PKGS := $(shell go list $(COMMON_GOFLAGS)  ./... | grep -v $(PROJECT)/vendor | grep -v $(PROJECT)/tests)
FILES := odo dist
TIMEOUT ?= 14400s

# Env variable TEST_EXEC_NODES is used to pass spec execution type
# (parallel or sequential) for ginkgo tests. To run the specs sequentially use
# TEST_EXEC_NODES=1, otherwise by default the specs are run in parallel on 4 ginkgo test node.
# NOTE: Any TEST_EXEC_NODES value greater than one runs the spec in parallel
# on the same number of ginkgo test nodes.
TEST_EXEC_NODES ?= 24

# Slow spec threshold for ginkgo tests. After this time (in second), ginkgo marks test as slow
SLOW_SPEC_THRESHOLD := 120

# Env variable GINKGO_TEST_ARGS is used to get control over enabling ginkgo test flags against each test target run.
# For example:
# To enable verbosity export or set env GINKGO_TEST_ARGS like "GINKGO_TEST_ARGS=-v"
GINKGO_TEST_ARGS ?=

# ODO_LOG_LEVEL sets the verbose log level for the make tests
export ODO_LOG_LEVEL ?= 4

# Env variable UNIT_TEST_ARGS is used to get control over enabling test flags along with go test.
# For example:
# To enable verbosity export or set env GINKGO_TEST_ARGS like "GINKGO_TEST_ARGS=-v"
UNIT_TEST_ARGS ?=

GINKGO_FLAGS_ALL = $(GINKGO_TEST_ARGS) -randomizeAllSpecs -slowSpecThreshold=$(SLOW_SPEC_THRESHOLD) -timeout $(TIMEOUT)

# Flags for tests that must not be run in parallel.
GINKGO_FLAGS_SERIAL = $(GINKGO_FLAGS_ALL) -nodes=1
# Flags for tests that may be run in parallel
GINKGO_FLAGS=$(GINKGO_FLAGS_ALL) -nodes=$(TEST_EXEC_NODES)


RUN_GINKGO = GOFLAGS='-mod=vendor' go run $(COMMON_GOFLAGS) github.com/onsi/ginkgo/ginkgo

default: bin

.PHONY: help
help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-26s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: bin
bin: ## build the odo binary
	go build ${BUILD_FLAGS} cmd/odo/odo.go

.PHONY: install
install:
	go install ${BUILD_FLAGS} ./cmd/odo/

.PHONY: validate
validate: gofmt check-fit check-vendor vet validate-vendor-licenses sec golint ## run all validation tests

.PHONY: gofmt
gofmt:
	./scripts/check-gofmt.sh

.PHONY: check-vendor
check-vendor:
	go mod verify

.PHONY: check-fit
check-fit:
	./scripts/check-fit.sh

.PHONY: validate-vendor-licenses
validate-vendor-licenses:
	go run $(COMMON_GOFLAGS) github.com/frapposelli/wwhrd check -q

.PHONY: golint
golint:
	golangci-lint run ./... --timeout 5m

.PHONY: lint
lint: ## golint errors are only recommendations
	golint $(PKGS)

.PHONY: vet
vet:
	go vet $(PKGS)

.PHONY: sec
sec:
	go run $(COMMON_GOFLAGS) github.com/securego/gosec/v2/cmd/gosec -severity medium -confidence medium -exclude G304,G204 -quiet  ./...

.PHONY: clean
clean:
	@rm -rf $(FILES)

.PHONY: goget-tools
goget-tools:
	mkdir -p $(shell go env GOPATH)/bin
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.30.0

.PHONY: goget-ginkgo
goget-ginkgo:
	@echo "This is no longer used."
	@echo "Ginkgo can be executed directly from this repository using command '$(RUN_GINKGO)'"

.PHONY: test-coverage
test-coverage: ## Run unit tests and collect coverage
	./scripts/generate-coverage.sh

.PHONY: cross
cross: ## compile for multiple platforms
	./scripts/cross-compile.sh $(CROSS_BUILD_FLAGS)

.PHONY: generate-cli-structure
generate-cli-structure:
	go run cmd/cli-doc/cli-doc.go structure

.PHONY: generate-cli-reference
generate-cli-reference:
	go run cmd/cli-doc/cli-doc.go reference > docs/cli-reference.adoc

# run make cross before this!
.PHONY: prepare-release
prepare-release: cross ## create gzipped binaries in ./dist/release/ for uploading to GitHub release page
	./scripts/prepare-release.sh

.PHONY: configure-installer-tests-cluster
configure-installer-tests-cluster:
	. ./scripts/configure-installer-tests-cluster.sh

.PHONY: configure-installer-tests-cluster-s390x
configure-installer-tests-cluster-s390x: ## configure cluster to run tests on s390x arch
	. ./scripts/configure-installer-tests-cluster-s390x.sh

.PHONY: configure-installer-tests-cluster-ppc64le
configure-installer-tests-cluster-ppc64le: ## configure cluster to run tests on ppc64le arch
	. ./scripts/configure-installer-tests-cluster-ppc64le.sh

.PHONY: configure-supported-311-is
configure-supported-311-is:
	. ./scripts/supported-311-is.sh

.PHONY: test
test:
	go test $(UNIT_TEST_ARGS) -race $(PKGS)

.PHONY: test-windows
test-windows:
	go test $(UNIT_TEST_ARGS)  $(PKGS)

.PHONY: test-generic
test-generic: ## Run generic integration tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo generic" tests/integration/

.PHONY: test-cmd-login-logout
test-cmd-login-logout: ## Run odo login and logout tests
	$(RUN_GINKGO) $(GINKGO_FLAGS_SERIAL) -focus="odo login and logout command tests" tests/integration/loginlogout/

.PHONY: test-cmd-link-unlink-4-cluster
test-cmd-link-unlink-4-cluster: ## Run link and unlink commnad tests against 4.x cluster
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo link and unlink commnad tests" tests/integration/

.PHONY: test-cmd-link-unlink-311-cluster
test-cmd-link-unlink-311-cluster: ## Run link and unlink command tests against 3.11 cluster
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo link and unlink command tests" tests/integration/servicecatalog/

.PHONY: test-cmd-service
test-cmd-service: ## Run odo service command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo service command tests" tests/integration/servicecatalog/

.PHONY: test-cmd-project
test-cmd-project: ## Run odo project command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS_SERIAL) -focus="odo project command tests" tests/integration/project/

.PHONY: test-cmd-app
test-cmd-app: ## Run odo app command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo app command tests" tests/integration/

.PHONY: test-cmd-cmp
test-cmd-cmp: ## Run odo component command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo component command tests" tests/integration/

.PHONY: test-cmd-cmp-sub
test-cmd-cmp-sub: ## Run odo component subcommands tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo sub component command tests" tests/integration/

.PHONY: test-cmd-pref-config
test-cmd-pref-config: ## Run odo preference and config command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo preference and config command tests" tests/integration/

.PHONY: test-cmd-push
test-cmd-push: ## Run odo push command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo push command tests" tests/integration/

.PHONY: test-plugin-handler
test-plugin-handler: ## Run odo plugin handler tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo plugin functionality" tests/integration/

.PHONY: test-cmd-devfile-catalog
test-cmd-devfile-catalog: ## Run odo catalog devfile command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile catalog command tests" tests/integration/devfile/

.PHONY: test-cmd-devfile-create
test-cmd-devfile-create: ## Run odo create devfile command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile create command tests" tests/integration/devfile/

.PHONY: test-cmd-devfile-push
test-cmd-devfile-push: ## Run odo push devfile command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile push command tests" tests/integration/devfile/

.PHONY: test-cmd-devfile-exec
test-cmd-devfile-exec: ## Run odo exec devfile command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile exec command tests" tests/integration/devfile/

.PHONY: test-cmd-devfile-status
test-cmd-devfile-status: ## Run odo status devfile command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile status command tests" tests/integration/devfile/

.PHONY: test-cmd-devfile-watch
test-cmd-devfile-watch: ## Run odo devfile watch command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile watch command tests" tests/integration/devfile/

.PHONY: test-cmd-devfile-app
test-cmd-devfile-app: ## Run odo devfile app command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile app command tests" tests/integration/devfile/

.PHONY: test-cmd-devfile-delete
test-cmd-devfile-delete: ## Run odo devfile delete command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile delete command tests" tests/integration/devfile/

.PHONY: test-cmd-devfile-registry
test-cmd-devfile-registry: ## Run odo devfile registry command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile registry command tests" tests/integration/devfile/

.PHONY: test-cmd-devfile-test
test-cmd-devfile-test: ## Run odo devfile test command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile test command tests" tests/integration/devfile/
	
.PHONY: test-cmd-storage
test-cmd-storage: ## Run odo storage command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo storage command tests" tests/integration/

.PHONY: test-cmd-url
test-cmd-url: ## Run odo url command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo url command tests" tests/integration/

.PHONY: test-cmd-devfile-url
test-cmd-devfile-url: ## Run odo url devfile command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile url command tests" tests/integration/devfile/

.PHONY: test-cmd-devfile-debug
test-cmd-devfile-debug: ## Run odo debug devfile command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile debug command tests" tests/integration/devfile/
	$(RUN_GINKGO) $(GINKGO_FLAGS_SERIAL) -focus="odo devfile debug command serial tests" tests/integration/devfile/debug/

.PHONY: test-cmd-devfile-storage
test-cmd-devfile-storage: ## Run odo storage devfile command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile storage command tests" tests/integration/devfile/

.PHONY: test-cmd-devfile-log
test-cmd-devfile-log: ## Run odo log devfile command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile log command tests" tests/integration/devfile/

.PHONY: test-cmd-devfile-env
test-cmd-devfile-env: ## Run odo env devfile command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile env command tests" tests/integration/devfile/

.PHONY: test-cmd-devfile-config
test-cmd-devfile-config: ## Run odo config devfile command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile config command tests" tests/integration/devfile/

.PHONY: test-cmd-watch
test-cmd-watch: ## Run odo watch command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo watch command tests" tests/integration/

.PHONY: test-cmd-debug
test-cmd-debug: ## Run odo debug command tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo debug command tests" tests/integration/
	$(RUN_GINKGO) $(GINKGO_FLAGS_SERIAL) -focus="odo debug command serial tests" tests/integration/debug/

# Service, link and login/logout command tests are not the part of this test run
.PHONY: test-integration
test-integration: ## Run command's integration tests irrespective of service catalog status in the cluster.
	$(RUN_GINKGO) $(GINKGO_FLAGS) tests/integration/
	$(RUN_GINKGO) $(GINKGO_FLAGS_SERIAL) tests/integration/debug/

.PHONY: test-integration-devfile
test-integration-devfile: ## Run devfile integration tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) tests/integration/devfile/
	$(RUN_GINKGO) $(GINKGO_FLAGS_SERIAL) tests/integration/devfile/debug/

# Only service and link command tests are the part of this test run
.PHONY: test-integration-service-catalog
test-integration-service-catalog: ## Run command's integration tests which are dependent on service catalog enabled cluster.
	$(RUN_GINKGO) $(GINKGO_FLAGS) tests/integration/servicecatalog/

.PHONY: test-e2e-beta
test-e2e-beta: ## Run core beta flow e2e tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo core beta flow" tests/e2escenarios/

.PHONY: test-e2e-java
test-e2e-java: ## Run java e2e tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo java e2e tests" tests/e2escenarios/

.PHONY: test-e2e-source
test-e2e-source: ## Run source e2e tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo source e2e tests" tests/e2escenarios/

.PHONY: test-e2e-images
test-e2e-images: ## Run supported images e2e tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo supported images e2e tests" tests/e2escenarios/

.PHONY: test-e2e-devfile
test-e2e-devfile: ## Run devfile e2e tests: odo devfile supported tests
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile supported tests" tests/e2escenarios/

.PHONY: test-e2e-all
test-e2e-all: ## Run all e2e test scenarios
	$(RUN_GINKGO) $(GINKGO_FLAGS) tests/e2escenarios/

# run make cross before this!
.PHONY: packages
packages: ## create deb and rpm packages using fpm in ./dist/pkgs/
	./scripts/create-packages.sh

# run 'make cross' and 'make packages' before this!
.PHONY: upload-packages
upload-packages: ## upload packages created by 'make packages' to bintray repositories
	./scripts/upload-packages.sh

.PHONY: vendor-update
vendor-update: ## Update vendoring
	go mod vendor

.PHONY: openshiftci-presubmit-unittests
openshiftci-presubmit-unittests:
	./scripts/openshiftci-presubmit-unittests.sh

.PHONY: test-operator-hub
test-operator-hub: ## Run OperatorHub tests
	$(RUN_GINKGO) $(GINKGO_FLAGS_SERIAL) -focus="odo service command tests" tests/integration/operatorhub/

.PHONY: test-cmd-devfile-describe
test-cmd-devfile-describe:
	$(RUN_GINKGO) $(GINKGO_FLAGS) -focus="odo devfile describe command tests" tests/integration/devfile/
