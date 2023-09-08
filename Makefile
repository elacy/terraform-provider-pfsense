BINARY_NAME=terraform-provider-pfsense
VERSION=0.1.0
OS=$(shell uname | tr '[:upper:]' '[:lower:]')
ARCH=$(shell go env GOARCH)
PLUGIN_DIR=~/.terraform.d/plugins/terraform.local/local/pfsense
PF_PATH=terraform.local/local/pfsense
TF_PROVIDER_DIR=~/.terraform/providers/$(PF_PATH)
TF_PLUGIN_DIR=~/.terraform.d/plugins/$(PF_PATH)
TF_PLUGIN_BINARY_DIR=$(TF_PLUGIN_DIR)/$(VERSION)/$(OS)_$(ARCH)
TF_PROVIDER_BINARY_DIR=$(TF_PROVIDER_DIR)/$(VERSION)/$(OS)_$(ARCH)

# Build the custom Terraform provider
build:
	@echo "Building the provider..."
	go build -o $(BINARY_NAME)

# Install the custom provider to the local Terraform plugins directory
local-install: build
	@echo "Installing the provider to local Terraform plugins directory..."
	rm -rf $(TF_PROVIDER_DIR)
	rm -rf $(TF_PLUGIN_DIR)
	mkdir -p $(TF_PROVIDER_BINARY_DIR)
	mkdir -p $(TF_PLUGIN_BINARY_DIR)
	cp $(BINARY_NAME) $(TF_PLUGIN_BINARY_DIR)/$(BINARY_NAME)
	chmod +x $(TF_PLUGIN_BINARY_DIR)/$(BINARY_NAME)
	cp $(BINARY_NAME) $(TF_PROVIDER_BINARY_DIR)/$(BINARY_NAME)
	chmod +x $(TF_PROVIDER_BINARY_DIR)/$(BINARY_NAME)
	@echo "Provider installed at $(TF_PLUGIN_BINARY_DIR) and $(TF_PROVIDER_BINARY_DIR)!"


.PHONY: build local-install
