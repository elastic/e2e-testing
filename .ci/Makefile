PROJECT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/..
VENV_DIR=$(PROJECT_DIR)/.venv
RUN_ID_FILE=$(PROJECT_DIR)/.ci/.runID
RUN_ID:=$(shell cat $(RUN_ID_FILE))

SSH_KEY ?= $(HOME)/.ssh/id_rsa
SSH_KEY_PUBLIC = $(SSH_KEY).pub

# Debian 10 AMD (see .e2e-platforms.yaml)
STACK_IMAGE=ami-0d90bed76900e679a
STACK_INSTANCE_TYPE=t3.xlarge
STACK_LABEL=debian_amd64
STACK_USER=admin
STACK_IP_ADDRESS ?= $(shell cat $(PROJECT_DIR)/.ci/.stack-host-ip)

# Debian 10 AMD (see .e2e-platforms.yaml)
NODE_IMAGE ?= ami-0d90bed76900e679a
NODE_INSTANCE_TYPE ?= t3.xlarge
NODE_LABEL ?= debian_amd64
NODE_SHELL_TYPE ?= sh
NODE_USER ?= admin
NODE_IP_ADDRESS ?= $(shell cat $(PROJECT_DIR)/.ci/.node-host-ip)

# Provider to be used in the test run
PROVIDER ?= remote

# Suite to run
SUITE ?= fleet
# Tags to run. Please check out the feature files
TAGS ?= fleet_mode

# Check that given variables are set and all have non-empty values,
# die with an error otherwise.
#
# Params:
#   1. Variable name(s) to test.
#   2. (optional) Error message to print.
check_defined = \
    $(strip $(foreach 1,$1, \
        $(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = \
    $(if $(value $1),, \
      $(error Undefined $1$(if $2, ($2))))

validate-aws:
ifndef AWS_ACCESS_KEY_ID
	@echo "You need to export AWS_ACCESS_KEY_ID to create AWS resources under that account"
	@exit 1
endif
ifndef AWS_SECRET_ACCESS_KEY
	@echo "You need to export AWS_SECRET_ACCESS_KEY to create AWS resources under that account"
	@exit 1
endif

# Create the run id if it does not exist
.runID:
	echo "$(shell uuidgen|cut -d'-' -f1)" > $(RUN_ID_FILE)

.PHONY: setup-env
setup-env:
	$(MAKE) .runID
	python3 -mvenv $(VENV_DIR)
	$(VENV_DIR)/bin/pip3 install ansible requests boto3 boto
	$(VENV_DIR)/bin/ansible-galaxy install -r $(PROJECT_DIR)/.ci/ansible/requirements.yml

.PHONY: destroy-env
destroy-env:
	rm -fr $(VENV_DIR)

.PHONY: list-platforms
list-platforms:
	@docker run --rm -i -w "/workdir" -v $(PROJECT_DIR)/.ci:/workdir mikefarah/yq:4 ".PLATFORMS | keys" .e2e-platforms.yaml

.PHONY: set-env-%
set-env-%:
	@$(PROJECT_DIR)/.ci/scripts/yq.sh "$*"

.PHONY: set-env-stack
set-env-stack:
	@$(PROJECT_DIR)/.ci/scripts/yq.sh "stack"

.PHONY: provision-stack
provision-stack: .runID validate-aws
	@:$(call check_defined, RUN_ID, You need to an unique RUN_ID. To create it please run 'make .runID' goal)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeInstanceType=$(STACK_INSTANCE_TYPE) nodeUser=$(STACK_USER)" \
		--extra-vars "runId=$(RUN_ID) nodeShellType=$(STACK_SHELL_TYPE) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t provision-stack

.PHONY: setup-stack
setup-stack: export TAGS = non-existing-tag
setup-stack: .runID validate-aws
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	@:$(call check_defined, STACK_IP_ADDRESS, IP address of the stack not defined)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeInstanceType=$(STACK_INSTANCE_TYPE) nodeUser=$(STACK_USER)" \
		--extra-vars "runId=$(RUN_ID) nodeShellType=$(STACK_SHELL_TYPE) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t setup-stack \
		-i $(STACK_IP_ADDRESS),

.PHONY: create-stack
create-stack: provision-stack setup-stack

.PHONY: destroy-stack
destroy-stack: validate-aws
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars="nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeUser=$(STACK_IMAGE)" \
		--extra-vars="runId=$(RUN_ID) nodeShellType=$(STACK_SHELL_TYPE) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t destroy
	rm -fr $(PROJECT_DIR)/.ci/.stack-host-ip

.PHONY: ssh-stack
ssh-stack:
	ssh -i $(SSH_KEY) $(STACK_USER)@$(STACK_IP_ADDRESS)

.PHONY: show-stack
show-stack:
	@echo "Stack IP address    : $(STACK_IP_ADDRESS)"
	@echo "Stack type          : $(STACK_LABEL)"
	@echo "Stack instance type : $(STACK_INSTANCE_TYPE)"
	@echo "Stack SSH user      : $(STACK_USER)"
	@echo "Stack Shell         : $(STACK_SHELL_TYPE)"

.PHONY: provision-node
provision-node: .runID validate-aws
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	@:$(call check_defined, STACK_IP_ADDRESS, IP address of the stack not defined)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "stackRunner=$(STACK_IP_ADDRESS) nodeLabel=$(NODE_LABEL) nodeImage=$(NODE_IMAGE) nodeInstanceType=$(NODE_INSTANCE_TYPE) nodeUser=$(NODE_USER)" \
		--extra-vars "runId=$(RUN_ID) nodeShellType=$(NODE_SHELL_TYPE) workspace=$(PROJECT_DIR)/ suite=$(SUITE) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t provision-node

.PHONY: setup-node
setup-node: .runID
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	@:$(call check_defined, STACK_IP_ADDRESS, IP address of the stack not defined)
	@:$(call check_defined, NODE_IP_ADDRESS, IP address of the test node not defined)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "stackRunner=$(STACK_IP_ADDRESS) nodeLabel=$(NODE_LABEL) nodeImage=$(NODE_IMAGE) nodeInstanceType=$(NODE_INSTANCE_TYPE) nodeUser=$(NODE_USER)" \
		--extra-vars "runId=$(RUN_ID) nodeShellType=$(NODE_SHELL_TYPE) workspace=$(PROJECT_DIR)/ suite=$(SUITE) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o ConnectTimeout=180s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t setup-node \
		-i $(NODE_IP_ADDRESS),

.PHONY: create-node
create-node: provision-node setup-node

.PHONY: destroy-node
destroy-node: validate-aws
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars="nodeLabel=$(NODE_LABEL) nodeImage=$(NODE_IMAGE) nodeUser=$(NODE_USER)" \
		--extra-vars="runId=$(RUN_ID) nodeShellType=$(NODE_SHELL_TYPE) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t destroy
	rm -fr $(PROJECT_DIR)/.ci/.node-host-ip

.PHONY: ssh-node
ssh-node:
	ssh -i $(SSH_KEY) $(NODE_USER)@$(NODE_IP_ADDRESS)

.PHONY: show-node
show-node:
	@echo "Node IP            : $(NODE_IP_ADDRESS)"
	@echo "Node type          : $(NODE_LABEL)"
	@echo "Node instance type : $(NODE_INSTANCE_TYPE)"
	@echo "Node SSH user      : $(NODE_USER)"
	@echo "Node Shell         : $(NODE_SHELL_TYPE)"

.PHONY: destroy-elastic-stack
destroy-elastic-stack:
	ssh -i $(SSH_KEY) $(STACK_USER)@$(STACK_IP_ADDRESS) 'sudo docker-compose -f /root/.op/compose/profiles/fleet/docker-compose.yml down --remove-orphans'

.PHONY: recreate-fleet-server
recreate-fleet-server: validate-aws
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	PROVIDER="remote" SUITE="$(SUITE)" TAGS="non-existent-tag" \
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/fleet-server.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeInstanceType=$(STACK_INSTANCE_TYPE) nodeUser=$(STACK_USER)" \
		--extra-vars "runId=$(RUN_ID) nodeShellType=$(STACK_SHELL_TYPE) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t fleet-server \
		-i $(STACK_IP_ADDRESS),
	ssh -t -i $(SSH_KEY) $(STACK_USER)@$(STACK_IP_ADDRESS) 'sudo bash "/home/$(STACK_USER)/e2e-testing/.ci/scripts/functional-test.sh"'

.PHONY: run-tests
run-tests: validate-aws
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	PROVIDER="$(PROVIDER)" SUITE="$(SUITE)" TAGS="$(TAGS)" REPORT_PREFIX="$(SUITE)_$(NODE_LABEL)_$(TAGS)" \
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/run-tests.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "stackRunner=$(STACK_IP_ADDRESS) nodeLabel=$(NODE_LABEL) nodeImage=$(NODE_IMAGE) nodeInstanceType=$(NODE_INSTANCE_TYPE) nodeUser=$(NODE_USER)" \
		--extra-vars "runId=$(RUN_ID) nodeShellType=$(NODE_SHELL_TYPE) workspace=$(PROJECT_DIR)/ suite=$(SUITE) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t run-tests \
		-i $(NODE_IP_ADDRESS),
	ssh -i $(SSH_KEY) $(NODE_USER)@$(NODE_IP_ADDRESS) 'sudo bash /home/$(NODE_USER)/e2e-testing/.ci/scripts/functional-test.sh \"$(TAGS)\"'

.PHONY: run-tests-win
run-tests-win: validate-aws
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	PROVIDER="$(PROVIDER)" SUITE="$(SUITE)" TAGS="$(TAGS)" REPORT_PREFIX="$(SUITE)_$(NODE_LABEL)_$(TAGS)" \
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/run-tests.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "stackRunner=$(STACK_IP_ADDRESS) nodeLabel=$(NODE_LABEL) nodeImage=$(NODE_IMAGE) nodeInstanceType=$(NODE_INSTANCE_TYPE) nodeUser=$(NODE_USER)" \
		--extra-vars "runId=$(RUN_ID) nodeShellType=$(NODE_SHELL_TYPE) workspace=$(PROJECT_DIR)/ suite=$(SUITE) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t run-tests \
		-i $(NODE_IP_ADDRESS),
	ssh -i $(SSH_KEY) $(NODE_USER)@$(NODE_IP_ADDRESS) 'powershell "C:\Users\$(NODE_USER)\e2e-testing\.ci\scripts\functional-test.ps1"'

# Make goal to run the tests for the MacOS platform. Used by the CI.
.PHONY: run-fleet-tests-macos
run-fleet-tests-macos:
	FORMAT="pretty,cucumber:fleet_mode.json,junit:fleet_mode.xml" \
	PROVIDER='remote' \
	TAGS="fleet_mode && install" \
	$(MAKE) -C "../e2e/_suites/fleet" functional-test
