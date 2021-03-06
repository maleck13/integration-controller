ORG=integreatly
NAMESPACE=integration-services
PROJECT=integration-controller
SHELL = /bin/bash
TAG = 0.0.1
PKG = github.com/integr8ly/integration-controller
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go -exec dirname {} \\; | sort | uniq")
CRD_NAME=integration

.PHONY: check-gofmt
check-gofmt:
	diff -u <(echo -n) <(gofmt -d `find . -type f -name '*.go' -not -path "./vendor/*"`)



.PHONY: test-unit
test-unit:
	@echo Running tests:
	go test -v -race -cover ./pkg/...

.PHONY: setup
setup:
	@echo Installing operator-sdk cli
	cd vendor/github.com/operator-framework/operator-sdk/commands/operator-sdk/ && go install .
	@echo Installing dep
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	@echo Installing errcheck
	@go get github.com/kisielk/errcheck
	@echo setup complete run make build deploy to build and deploy the operator to a local cluster


.PHONY: build
build-image:
	operator-sdk build quay.io/${ORG}/${PROJECT}:${TAG}

.PHONY: run
run:
	operator-sdk up local --namespace=${NAMESPACE} --operator-flags="--resync=5 --log-level=debug"

.PHONY: generate
generate:
	operator-sdk generate k8s

compile:
	go build -o=${PROJECT} ./cmd/${PROJECT}

.PHONY: check
check: check-gofmt test-unit
	@echo errcheck
	@errcheck -ignoretests $$(go list ./...)
	@echo go vet
	@go vet ./...

.PHONY: install
install: install_crds
	-oc new-project $(NAMESPACE)
	-kubectl create --insecure-skip-tls-verify -f deploy/rbac.yaml -n $(NAMESPACE)

.PHONY: install_crds
install_crds:
	-kubectl create -f deploy/crd.yaml


.PHONY: uninstall
uninstall:
	-kubectl delete role ${PROJECT} -n $(NAMESPACE)
	-kubectl delete rolebinding default-account-${PROJECT} -n $(NAMESPACE)
	-kubectl delete crd ${CRD_NAME}s.integr8ly.org
	-kubectl delete namespace $(NAMESPACE)


.PHONY: create-examples
create-examples:
		-kubectl create -f deploy/examples/${CRD_NAME}.json -n $(NAMESPACE)
