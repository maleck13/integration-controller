#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

vendor/k8s.io/code-generator/generate-groups.sh \
deepcopy \
github.com/integr8ly/integration-controller/pkg/generated \
github.com/integr8ly/integration-controller/pkg/apis \
integration:v1alpha1 \
--go-header-file "./tmp/codegen/boilerplate.go.txt"

vendor/k8s.io/code-generator/generate-groups.sh \
deepcopy \
github.com/integr8ly/integration-controller/pkg/generated \
github.com/integr8ly/integration-controller/pkg/apis \
enmasse:v1 \
--go-header-file "./tmp/codegen/boilerplate.go.txt"

vendor/k8s.io/code-generator/generate-groups.sh \
deepcopy \
github.com/integr8ly/integration-controller/pkg/generated \
github.com/integr8ly/integration-controller/pkg/apis \
syndesis:v1alpha1 \
--go-header-file "./tmp/codegen/boilerplate.go.txt"
