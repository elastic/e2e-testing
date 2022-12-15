PROJECT_DIR:=$(abspath $(dir $(lastword $(MAKEFILE_LIST)))/../)
VENV_DIR=$(PROJECT_DIR)/.venv
VENV_BIN_DIR=$(VENV_DIR)/bin
ANSIBLE_GALAXY=$(VENV_BIN_DIR)/ansible-galaxy
ANSIBLE_PLAYBOOK=$(VENV_BIN_DIR)/ansible-playbook
PIP3=$(VENV_BIN_DIR)/pip3

RUN_ID_FILE=$(PROJECT_DIR)/.ci/.runID
RUN_ID ?= $(shell cat $(RUN_ID_FILE))

SSH_KEY ?= $(HOME)/.ssh/id_rsa
SSH_KEY_PUBLIC = $(SSH_KEY).pub
SSH_OPTS=-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null
SSH_OPTS_EXTENDED=-o TCPKeepAlive=yes -o ServerAliveInterval=30 -o ServerAliveCountMax=200 $(SSH_OPTS)

# Debian 10 AMD (see .e2e-platforms.yaml)
STACK_IMAGE=ami-0d90bed76900e679a
STACK_INSTANCE_TYPE=t3.xlarge
STACK_LABEL=debian_amd64
STACK_USER=admin
STACK_IP_ADDRESS ?= $(shell cat $(PROJECT_DIR)/.ci/.stack-host-ip)
STACK_INSTANCE_ID ?= stack_$(STACK_LABEL)_$(RUN_ID)

# Debian 10 AMD (see .e2e-platforms.yaml)
NODE_IMAGE ?= ami-0d90bed76900e679a
NODE_INSTANCE_TYPE ?= t3.xlarge
NODE_LABEL ?= debian_amd64
NODE_SHELL_TYPE ?= sh
NODE_USER ?= admin
NODE_IP_ADDRESS ?= $(shell cat $(PROJECT_DIR)/.ci/.node-host-ip)
NODE_INSTANCE_ID ?= $(NODE_LABEL)_$(RUN_ID)

# Extra labels to be used in the Ansible execution
LABELS_STRING ?= 

# Provider to be used in the test run
PROVIDER ?= remote

# Suite to run
SUITE ?= fleet
# Tags to run. Please check out the feature files
TAGS ?= fleet_mode

SHELL = /bin/bash
MAKEFLAGS += --silent --no-print-directory
.SHELLFLAGS = -ec

export ANSIBLE_CONFIG := $(CURDIR)/ansible/ansible.cfg

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

@:$(call check_defined, AWS_ACCESS_KEY_ID, You need to export AWS_ACCESS_KEY_ID to create AWS resources under that account)
@:$(call check_defined, AWS_SECRET_ACCESS_KEY, You need to export AWS_SECRET_ACCESS_KEY to create AWS resources under that account)

# Create the run id if it does not exist
.runID:
	echo "$(shell uuidgen|cut -d'-' -f1)" > $(RUN_ID_FILE)

.PHONY: show-env
show-env:
	@source $(VENV_DIR)/bin/activate; \
	echo "PROJECT_DIR: $(PROJECT_DIR)"; \
	echo "VENV_DIR: $(VENV_DIR)"; \
	echo "VENV_BIN_DIR: $(VENV_BIN_DIR)"; \
	echo "ANSIBLE: $$(ansible --version)"; \
	echo "Python: $$(python --version)"; \
	echo "RUN_ID: $(RUN_ID)"; \
	echo "STACK_VERSION: $(STACK_VERSION)"; \
	echo "SUITE: $(SUITE)"; \
	echo "TAGS: $(TAGS)"; \
	echo "PROVIDER: $(PROVIDER)";

.PHONY: setup-env
setup-env:
	$(MAKE) .runID
	python3 -mvenv $(VENV_DIR)
	$(PIP3) -q install wheel
	$(PIP3) install -r $(PROJECT_DIR)/.ci/ansible/requirements.txt
	$(ANSIBLE_GALAXY) install -r $(PROJECT_DIR)/.ci/ansible/requirements.yml

.PHONY: destroy-env
destroy-env:
	rm -fr $(VENV_DIR)

.PHONY: list-platforms
list-platforms:
	@docker run --rm -i -w "/workdir" -v $(PROJECT_DIR)/.ci:/workdir mikefarah/yq:4 ".PLATFORMS | keys" .e2e-platforms.yaml

# Create the env file for the target platform
.PHONY: set-env-%
set-env-%:
	@$(PROJECT_DIR)/.ci/scripts/yq.sh "$*"

.PHONY: provision-stack
provision-stack: .runID show-env
	@:$(call check_defined, RUN_ID, You need to an unique RUN_ID. To create it please run 'make .runID' goal)
	source $(VENV_DIR)/bin/activate; \
	$(ANSIBLE_PLAYBOOK) $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeInstanceType=$(STACK_INSTANCE_TYPE) nodeUser=$(STACK_USER)" \
		--extra-vars "runId=$(RUN_ID) instanceID=$(STACK_INSTANCE_ID) nodeShellType=$(STACK_SHELL_TYPE) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='$(SSH_OPTS)' \
		-t provision-stack

.PHONY: setup-stack
setup-stack: export TAGS = non-existing-tag
setup-stack: .runID show-env
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	@:$(call check_defined, STACK_IP_ADDRESS, IP address of the stack not defined)
	source $(VENV_DIR)/bin/activate; \
	$(ANSIBLE_PLAYBOOK) $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeInstanceType=$(STACK_INSTANCE_TYPE) nodeUser=$(STACK_USER)" \
		--extra-vars "runId=$(RUN_ID) nodeShellType=$(STACK_SHELL_TYPE) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='$(SSH_OPTS)' \
		-t setup-stack \
		-i $(STACK_IP_ADDRESS),

.PHONY: create-stack
create-stack: provision-stack setup-stack start-elastic-stack

.PHONY: destroy-stack
destroy-stack: show-env
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	source $(VENV_DIR)/bin/activate; \
	$(ANSIBLE_PLAYBOOK) $(PROJECT_DIR)/.ci/ansible/teardown.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars="$(LABELS_STRING) nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeUser=$(STACK_IMAGE)" \
		--extra-vars="runId=$(RUN_ID) instanceID=$(STACK_INSTANCE_ID) nodeShellType=$(STACK_SHELL_TYPE) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='$(SSH_OPTS)'
	# rm -fr $(PROJECT_DIR)/.ci/.stack-host-ip

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
provision-node: .runID set-env-$(NODE_LABEL)
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	@:$(call check_defined, STACK_IP_ADDRESS, IP address of the stack not defined)
	source $(VENV_DIR)/bin/activate; \
	. $(PROJECT_DIR)/.ci/.env-$(NODE_LABEL) && $(ANSIBLE_PLAYBOOK) $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) stackRunner=$(STACK_IP_ADDRESS) nodeLabel=$${NODE_LABEL} nodeImage=$${NODE_IMAGE} nodeInstanceType=$${NODE_INSTANCE_TYPE} nodeUser=$${NODE_USER}" \
		--extra-vars "runId=$(RUN_ID) instanceID=$(NODE_INSTANCE_ID) nodeShellType=$${NODE_SHELL_TYPE} workspace=$(PROJECT_DIR)/ suite=$(SUITE) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='$(SSH_OPTS)' \
		-t provision-node

.PHONY: setup-node
setup-node: .runID set-env-$(NODE_LABEL)
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	@:$(call check_defined, STACK_IP_ADDRESS, IP address of the stack not defined)
	@:$(call check_defined, NODE_IP_ADDRESS, IP address of the test node not defined)
	source $(VENV_DIR)/bin/activate; \
	. $(PROJECT_DIR)/.ci/.env-$(NODE_LABEL) && $(ANSIBLE_PLAYBOOK) $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) stackRunner=$(STACK_IP_ADDRESS) nodeLabel=$${NODE_LABEL} nodeImage=$${NODE_IMAGE} nodeInstanceType=$${NODE_INSTANCE_TYPE} nodeUser=$${NODE_USER}" \
		--extra-vars "runId=$(RUN_ID) nodeShellType=$${NODE_SHELL_TYPE} workspace=$(PROJECT_DIR)/ suite=$(SUITE) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o ConnectTimeout=180s $(SSH_OPTS)' \
		-t setup-node \
		-i $(NODE_IP_ADDRESS),

.PHONY: create-node
create-node: provision-node setup-node

.PHONY: destroy-node
destroy-node: set-env-$(NODE_LABEL)
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	source $(VENV_DIR)/bin/activate; \
	. $(PROJECT_DIR)/.ci/.env-$(NODE_LABEL) && $(ANSIBLE_PLAYBOOK) $(PROJECT_DIR)/.ci/ansible/teardown.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars="$(LABELS_STRING) nodeLabel=$${NODE_LABEL} nodeImage=$${NODE_IMAGE} nodeUser=$${NODE_USER}" \
		--extra-vars="runId=$(RUN_ID) instanceID=$(NODE_INSTANCE_ID) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='$(SSH_OPTS)'
	rm -fr $(PROJECT_DIR)/.ci/.node-host-ip

.PHONY: fetch-test-reports
fetch-test-reports: .runID set-env-$(NODE_LABEL)
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	@:$(call check_defined, NODE_IP_ADDRESS, IP address of the test node not defined)
	source $(VENV_DIR)/bin/activate; \
	. $(PROJECT_DIR)/.ci/.env-$(NODE_LABEL) && $(ANSIBLE_PLAYBOOK) $(PROJECT_DIR)/.ci/ansible/fetch-test-reports.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) nodeLabel=$${NODE_LABEL} nodeImage=$${NODE_IMAGE} nodeInstanceType=$${NODE_INSTANCE_TYPE} nodeUser=$${NODE_USER}" \
		--extra-vars "runId=$(RUN_ID) nodeShellType=$${NODE_SHELL_TYPE} workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='$(SSH_OPTS)' \
		-t fetch-reports \
		-i $(NODE_IP_ADDRESS),

.PHONY: ssh-node
ssh-node: set-env-$(NODE_LABEL)
	. $(PROJECT_DIR)/.ci/.env-$(NODE_LABEL) && ssh -i $(SSH_KEY) $${NODE_USER}@$(NODE_IP_ADDRESS)

.PHONY: show-node
show-node: set-env-$(NODE_LABEL)
	@. $(PROJECT_DIR)/.ci/.env-$(NODE_LABEL) && \
		echo "Node IP            : $(NODE_IP_ADDRESS)" && \
		echo "Node type          : $${NODE_LABEL}" && \
		echo "Node instance type : $${NODE_INSTANCE_TYPE}" && \
		echo "Node SSH user      : $${NODE_USER}" && \
		echo "Node Shell         : $${NODE_SHELL_TYPE}"

.PHONY: destroy-elastic-stack
destroy-elastic-stack:
	ssh $(SSH_OPTS_EXTENDED) -i $(SSH_KEY) $(STACK_USER)@$(STACK_IP_ADDRESS) 'sudo docker-compose -f /root/.op/compose/profiles/fleet/docker-compose.yml down --remove-orphans'

.PHONY: start-elastic-stack
start-elastic-stack: show-env
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	source $(VENV_DIR)/bin/activate; \
	PROVIDER="remote" SUITE="$(SUITE)" TAGS="non-existent-tag" \
	$(ANSIBLE_PLAYBOOK) $(PROJECT_DIR)/.ci/ansible/elastic-stack.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeInstanceType=$(STACK_INSTANCE_TYPE) nodeUser=$(STACK_USER)" \
		--extra-vars "runId=$(RUN_ID) nodeShellType=$(STACK_SHELL_TYPE) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='$(SSH_OPTS)' \
		-t elastic-stack \
		-i $(STACK_IP_ADDRESS),
	ssh $(SSH_OPTS_EXTENDED) -i $(SSH_KEY) $(STACK_USER)@$(STACK_IP_ADDRESS) 'sudo bash "/home/$(STACK_USER)/e2e-testing/.ci/scripts/functional-test.sh"'

.PHONY: recreate-fleet-server
recreate-fleet-server:
	@echo "This command has been deprecated. Please use 'start-elastic-stack' instead"
	$(MAKE) start-elastic-stack

.PHONY: run-tests
run-tests: set-env-$(NODE_LABEL) show-env
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	source $(VENV_DIR)/bin/activate; \
	. $(PROJECT_DIR)/.ci/.env-$(NODE_LABEL) && PROVIDER="$(PROVIDER)" SUITE="$(SUITE)" TAGS="$(TAGS)" REPORT_PREFIX="$(SUITE)_$${NODE_LABEL}_$(TAGS)" \
	$(ANSIBLE_PLAYBOOK) $(PROJECT_DIR)/.ci/ansible/run-tests.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) stackRunner=$(STACK_IP_ADDRESS) nodeLabel=$${NODE_LABEL} nodeImage=$${NODE_IMAGE} nodeInstanceType=$${NODE_INSTANCE_TYPE} nodeUser=$${NODE_USER}" \
		--extra-vars "runId=$(RUN_ID) nodeShellType=$${NODE_SHELL_TYPE} workspace=$(PROJECT_DIR)/ suite=$(SUITE) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='$(SSH_OPTS)' \
		-t run-tests \
		-i $(NODE_IP_ADDRESS),
	. $(PROJECT_DIR)/.ci/.env-$(NODE_LABEL) && ssh $(SSH_OPTS_EXTENDED) -i $(SSH_KEY) $${NODE_USER}@$(NODE_IP_ADDRESS) "sudo bash /home/$${NODE_USER}/e2e-testing/.ci/scripts/functional-test.sh \"$(TAGS)\""

.PHONY: run-tests-win
run-tests-win: set-env-$(NODE_LABEL) show-env
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	source $(VENV_DIR)/bin/activate; \
	. $(PROJECT_DIR)/.ci/.env-$(NODE_LABEL) && PROVIDER="$(PROVIDER)" SUITE="$(SUITE)" TAGS="$(TAGS)" REPORT_PREFIX="$(SUITE)_$${NODE_LABEL}_$(TAGS)" \
	$(ANSIBLE_PLAYBOOK) $(PROJECT_DIR)/.ci/ansible/run-tests.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) stackRunner=$(STACK_IP_ADDRESS) nodeLabel=$${NODE_LABEL} nodeImage=$${NODE_IMAGE} nodeInstanceType=$${NODE_INSTANCE_TYPE} nodeUser=$${NODE_USER}" \
		--extra-vars "runId=$(RUN_ID) nodeShellType=$${NODE_SHELL_TYPE} workspace=$(PROJECT_DIR)/ suite=$(SUITE) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='$(SSH_OPTS)' \
		-t run-tests \
		-i $(NODE_IP_ADDRESS),
	. $(PROJECT_DIR)/.ci/.env-$(NODE_LABEL) && ssh $(SSH_OPTS_EXTENDED) -i $(SSH_KEY) $${NODE_USER}@$(NODE_IP_ADDRESS) "powershell \"C:/Users/$${NODE_USER}/e2e-testing/.ci/scripts/functional-test.ps1\""
