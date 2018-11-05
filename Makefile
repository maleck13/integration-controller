ORG=integreatly
NAMESPACE=integration-services
USER_NAMESPACE=
PROJECT=integration-controller
SHELL = /bin/bash
TAG = 0.0.1
PKG = github.com/integr8ly/integration-controller
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go -exec dirname {} \\; | sort | uniq")
CRD_NAME=integration
SA_TOKEN ?= $(oc sa get-token integration-controller)
PACKAGES ?= $(go list ./... | grep -v /vendor/)


.PHONY: check-gofmt
check-gofmt:
	diff -u <(echo -n) <(gofmt -d `find . -type f -name '*.go' -not -path "./vendor/*"`)



.PHONY: test-unit
test-unit:
	@echo Running tests:
	go test -v -race -cover ./pkg/...

.PHONY: setup
setup:
	@echo Installing dep
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	@echo Installing errcheck
	@go get github.com/kisielk/errcheck
	@echo setup complete run make build deploy to build and deploy the operator to a local cluster

.PHONY: k8sservice-integration
k8sservice-integration:
#install syndesis in a namesapce
	-oc create -f https://raw.githubusercontent.com/syndesisio/syndesis/release-work-1.5.0.fuse-720/install/operator/deploy/syndesis-crd.yml
	-oc create -f https://raw.githubusercontent.com/syndesisio/syndesis/release-work-1.5.0.fuse-720/install/operator/deploy/syndesis-operator.yml -n ${NAMESPACE}
	-oc create -f https://raw.githubusercontent.com/syndesisio/syndesis/release-work-1.5.0.fuse-720/install/operator/deploy/syndesis.yml -n ${NAMESPACE}
	@echo 'run oc port-forward <syndesis server pod> 8145:8080'

.PHONY: build-image
build-image:
	operator-sdk build quay.io/${ORG}/${PROJECT}:${TAG}

.PHONY: run
run:
	oc project ${NAMESPACE}
	export USER_NAMESPACES=${USER_NAMESPACE} && operator-sdk up local --namespace=${NAMESPACE} --operator-flags="--resync=20 --log-level=debug --sa-token=${SA_TOKEN}"

.PHONY: generate
generate:
	operator-sdk generate k8s
	@go generate ./...

compile:
	go build -o=${PROJECT} ./cmd/${PROJECT}

.PHONY: check
check: check-gofmt test-unit
	@echo errcheck
	@errcheck -ignoretests $$(go list ./...)
	@echo go vet
	@go vet ./pkg/...

.PHONY: deploy
deploy:
	-oc delete -f deploy/operator-deploy.yaml -n $(NAMESPACE)
	-oc create -f deploy/operator-deploy.yaml -n $(NAMESPACE)

.PHONY: install
install: install_crds
	-oc new-project $(NAMESPACE)
	-oc new-project $(USER_NAMESPACE)
	-oc create -f deploy/enmasse/enmasse-cluster-role.yaml
	-oc create -f deploy/applications/route-services-integration-cluster-role.yaml
	-oc create -f deploy/sa.yaml -n $(NAMESPACE)
	-oc create -f deploy/rbac.yaml -n $(NAMESPACE)
	-cat deploy/enmasse/enmasse-role-binding.yaml | sed -e 's/FUSE_NAMESPACE/$(NAMESPACE)/g' | oc create -n enmasse -f -
	-cat deploy/applications/route-services-integration-role-binding.yaml | sed -e 's/FUSE_NAMESPACE/$(NAMESPACE)/g' | oc create -n $(USER_NAMESPACE) -f -

.PHONY: install_crds
install_crds:
	-oc create -f deploy/crd.yaml


.PHONY: uninstall
uninstall:
	-oc delete -f deploy/enmasse/enmasse-cluster-role.yaml
	-oc delete -f deploy/applications/route-services-integration-cluster-role.yaml
	-oc delete -f deploy/sa.yaml -n $(NAMESPACE)
	-oc delete -f deploy/rbac.yaml -n $(NAMESPACE)
	-cat deploy/enmasse/enmasse-role-binding.yaml | sed -e 's/FUSE_NAMESPACE/$(NAMESPACE)/g' | oc delete -n enmasse -f -
	-cat deploy/applications/route-services-integration-cluster-role.yaml | sed -e 's/FUSE_NAMESPACE/$(NAMESPACE)/g' | oc delete -n $(NAMESPACE) -f -
	-oc delete -f deploy/crd.yaml
	-oc delete rolebindings -l for=integration-controller -n enmasse
	-oc delete namespace $(NAMESPACE)


.PHONY: create-examples
create-examples:
		-oc create -f deploy/examples/${CRD_NAME}.json -n $(NAMESPACE)
