##############################################################
# k-o11y unified build and push command
##############################################################
.PHONY: build-and-push
build-and-push: ## Build and push docker image (Usage: make build-and-push)
	@echo "Select package to build:"
	@echo "  1. core"
	@echo "  2. hub (signoz)"
	@read -p "Enter choice (1 or 2): " package_choice; \
	if [ "$$package_choice" = "1" ]; then \
		read -p "Enter TAG version (e.g., 0.1.3): " tag_input; \
		echo ">> Building core package with TAG=$$tag_input"; \
		cd packages/core && $(MAKE) core-build-and-push TAG=$$tag_input; \
	elif [ "$$package_choice" = "2" ]; then \
		read -p "Enter TAG version (e.g., 0.1.20): " tag_input; \
		echo ">> Building hub (signoz) package with TAG=$$tag_input"; \
		cd packages/signoz && $(MAKE) o11y-build-and-push TAG=$$tag_input; \
	else \
		echo "Invalid choice. Please enter 1 or 2."; \
		exit 1; \
	fi

.PHONY: help
help: ## Display available commands
	@echo "Available commands:"
	@echo "  make build-and-push     - Local build and push for core or hub package"
	@echo "  make ci-build-and-push  - Trigger CI build and push via GitHub Actions"

.PHONY: ci-build-and-push
ci-build-and-push: ## Build and push via GitHub Actions CI (Usage: make ci-build-and-push)
	@echo "Select package to build (via CI):"
	@echo "  1. core"
	@echo "  2. hub (signoz)"
	@echo "  3. both"
	@current_branch=$$(git rev-parse --abbrev-ref HEAD); \
	read -p "Enter choice (1, 2, or 3): " package_choice; \
	read -p "Enter TAG version (e.g., 0.1.3): " tag_input; \
	read -p "Enter branch (default: $$current_branch): " branch_input; \
	if [ -z "$$branch_input" ]; then \
		branch_input=$$current_branch; \
	fi; \
	if [ "$$package_choice" = "1" ]; then \
		package_name="core"; \
	elif [ "$$package_choice" = "2" ]; then \
		package_name="hub"; \
	elif [ "$$package_choice" = "3" ]; then \
		package_name="both"; \
	else \
		echo "Invalid choice. Please enter 1, 2, or 3."; \
		exit 1; \
	fi; \
	echo ">> Triggering CI build for $$package_name with TAG=$$tag_input on branch $$branch_input"; \
	gh api repos/:owner/:repo/actions/workflows/build-and-push.yaml/dispatches \
		-f ref=$$branch_input \
		-f "inputs[package]=$$package_name" \
		-f "inputs[tag]=$$tag_input"; \
	echo ">> CI build triggered! Check progress:"; \
	echo "   gh run list --workflow=build-and-push.yaml --limit=1"
