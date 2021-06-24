# Copyright 2021 The Nuclio Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#	 http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


GOPATH ?= $(shell go env GOPATH)


.PHONY: fmt
fmt: ## Code formatting
	gofmt -s -w .

.PHONY: lint
lint: modules  ## Code linting
	@echo Installing linters...
	@test -e $(GOPATH)/bin/impi || \
		(mkdir -p $(GOPATH)/bin && \
		curl -s https://api.github.com/repos/pavius/impi/releases/latest \
		| grep -i "browser_download_url.*impi.*$(OS_NAME)" \
		| cut -d : -f 2,3 \
		| tr -d \" \
		| wget -O $(GOPATH)/bin/impi -qi - \
		&& chmod +x $(GOPATH)/bin/impi)

	@test -e $(GOPATH)/bin/golangci-lint || \
	  	(curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.36.0)

	@echo Verifying imports...
	$(GOPATH)/bin/impi \
		--local github.com/nuclio/zap/ \
		--scheme stdLocalThirdParty \
		./...

	@echo Linting...
	$(GOPATH)/bin/golangci-lint run -v
	@echo Done.


.PHONY: test-unit
test-unit: modules ## Run unit tests
	go test -v ./... -short


## MISC


.PHONY: modules
modules: ensure-gopath  ## Download go module packages
	@echo Getting go modules
	@go mod download


.PHONY: ensure-gopath
ensure-gopath:  ## Ensure GOPATH env is set
ifndef GOPATH
	$(error GOPATH must be set)
endif

.PHONY: help
help: ## Display available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
