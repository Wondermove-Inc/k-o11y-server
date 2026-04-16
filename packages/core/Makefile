##############################################################
# k-o11y core build and push command
##############################################################
.PHONY: core-build-and-push
core-build-and-push: ## Build core docker image and push to registry (Usage: make core-build-and-push)
	@if [ -z "$(TAG)" ]; then \
		read -p "Enter TAG version (e.g., 0.1.3): " tag_input; \
		$(MAKE) core-build-and-push TAG=$$tag_input; \
	else \
		echo ">> Building and pushing core image with tag: $(TAG)"; \
		docker build -t <YOUR_REGISTRY>/observability/core:$(TAG) \
			-f deployments/Dockerfile \
			--platform linux/amd64 .; \
		echo ">> Build completed. Pushing to registry..."; \
		docker push <YOUR_REGISTRY>/observability/core:$(TAG); \
		echo ">> Successfully pushed <YOUR_REGISTRY>/observability/core:$(TAG)"; \
	fi


##############################################################
# GitHub Action workflow trigger
##############################################################
# Makefile for triggering GitHub Action workflows

######################## example ######################################
########### $ make trigger CUSTOM_TAG=v1.0.0 ##########################
#######################################################################

# Default values for parameters
# CSP ?= EKS
CUSTOM_TAG ?= latest

.PHONY: check-gh install-gh login-gh trigger-workflow

# Main target to trigger the workflow
trigger: check-gh trigger-workflow

# Check if gh CLI is installed
check-gh:
	@if ! command -v gh &> /dev/null; then \
		echo "GitHub CLI not found. Installing..."; \
		$(MAKE) install-gh; \
	else \
		echo "GitHub CLI found!"; \
	fi
	@if ! gh auth status &> /dev/null; then \
		echo "GitHub CLI not logged in. Please login:"; \
		$(MAKE) login-gh; \
	else \
		echo "GitHub CLI already authenticated!"; \
	fi

# Install GitHub CLI based on the OS
install-gh:
	@echo "Detecting operating system..."
	@if [ "$$(uname)" = "Darwin" ]; then \
		echo "Installing GitHub CLI for macOS using Homebrew..."; \
		brew install gh; \
	elif [ "$$(uname)" = "Linux" ]; then \
		echo "Installing GitHub CLI for Linux..."; \
		curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
		&& sudo chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
		&& echo "deb [arch=$$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
		&& sudo apt update \
		&& sudo apt install gh -y; \
	elif [ "$$(uname -o 2>/dev/null)" = "Msys" ] || [ "$$(uname -o 2>/dev/null)" = "Windows" ]; then \
		echo "Please install GitHub CLI for Windows from https://cli.github.com/"; \
		exit 1; \
	else \
		echo "Unsupported operating system. Please install GitHub CLI manually from https://cli.github.com/"; \
		exit 1; \
	fi

# Login to GitHub CLI
login-gh:
	@echo "Logging in to GitHub CLI..."
	@gh auth login

# Trigger the GitHub workflow
trigger-workflow:
	@echo "Triggering GitHub workflow with CUSTOM_TAG=$(CUSTOM_TAG)..."
	# @echo "Triggering GitHub workflow with CSP=$(CSP) and CUSTOM_TAG=$(CUSTOM_TAG)..."
	# @gh workflow run makefile-build-push.yaml -f CSP=$(CSP) -f CUSTOM_TAG=$(CUSTOM_TAG)
	@gh workflow run makefile-build-push.yaml -f CUSTOM_TAG=$(CUSTOM_TAG)
	@echo "Workflow triggered successfully!"