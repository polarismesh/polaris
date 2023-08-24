# Tencent is pleased to support the open source community by making Polaris available.
#
# Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
#
# Licensed under the BSD 3-Clause License (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://opensource.org/licenses/BSD-3-Clause
#
# Unless required by applicable law or agreed to in writing, software distributed
# under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
# CONDITIONS OF ANY KIND, either express or implied. See the License for the
# specific language governing permissions and limitations under the License.

# VERSION defines the project version for the build.
# Update this value when you upgrade the version file of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the build target (e.g make build VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= $(shell cat version 2>/dev/null)

# IMAGE_TAG defines the image tag for the docker build.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the IMAGE_TAG as arg of the build-docker target (e.g make build-docker IMAGE_TAG=v0.0.2)
# - use environment variables to overwrite this value (e.g export IMAGE_TAG=v0.0.2)
IMAGE_TAG ?= $(VERSION)

ARCH ?= "amd64"

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


##@ Build

.PHONY: build
build: ## Build binary and tarball.
	bash ./release/build.sh $(VERSION) $(ARCH)

.PHONY: build-docker
build-docker: ## Build polaris-server docker images.
	bash ./release/build_docker.sh $(IMAGE_TAG)

.PHONY: clean
clean: ## Clean polaris-server make data.
	@rm -rf polaris-server-release_*
	@rm -rf polaris-server-arm64
	@rm -rf polaris-server-amd64
