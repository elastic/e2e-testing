PROJECT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/..
VENV_DIR=$(PROJECT_DIR)/.venv
RUN_ID_FILE=$(PROJECT_DIR)/.ci/.runID
RUN_ID ?= $(shell cat $(RUN_ID_FILE))

SSH_KEY ?= $(HOME)/.ssh/id_rsa
SSH_KEY_PUBLIC = $(SSH_KEY).pub

# Debian 10 AMD (see .e2e-platforms.yaml)
STACK_IMAGE=ami-0d90bed76900e679a
STACK_INSTANCE_TYPE=c5.4xlarge
STACK_LABEL=debian_amd64
STACK_USER=admin
STACK_IP_ADDRESS ?= $(shell cat $(PROJECT_DIR)/.ci/.stack-host-ip)

# Debian 10 AMD (see .e2e-platforms.yaml)
NODE_IMAGE ?= ami-0d90bed76900e679a
NODE_INSTANCE_TYPE ?= c5.4xlarge
NODE_LABEL ?= debian_amd64
NODE_USER ?= admin
NODE_IP_ADDRESS ?= $(shell cat $(PROJECT_DIR)/.ci/.node-host-ip)

# Extra labels to be used in the Ansible execution
LABELS_STRING ?= 

# Provider to be used in the test run
PROVIDER ?= remote

# Suite to run
SUITE ?= fleet
# Tags to run. Please check out the feature files
TAGS ?= fleet_mode_agent

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

.PHONY: setup-env
setup-env:
	$(MAKE) .runID
	python3 -mvenv $(VENV_DIR)
	$(VENV_DIR)/bin/pip3 install ansible requests boto3 boto
	$(VENV_DIR)/bin/ansible-galaxy install -r $(PROJECT_DIR)/.ci/ansible/requirements.yml

.PHONY: destroy-env
destroy-env:
	rm -fr $(VENV_DIR)

.PHONY: provision-stack
provision-stack: .runID
	@:$(call check_defined, RUN_ID, You need to an unique RUN_ID. To create it please run 'make .runID' goal)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeInstanceType=$(STACK_INSTANCE_TYPE) nodeUser=$(STACK_USER)" \
		--extra-vars "runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t provision-stack

.PHONY: setup-stack
setup-stack: export TAGS = non-existing-tag
setup-stack: .runID
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	@:$(call check_defined, STACK_IP_ADDRESS, IP address of the stack not defined)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeInstanceType=$(STACK_INSTANCE_TYPE) nodeUser=$(STACK_USER)" \
		--extra-vars "runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t setup-stack \
		-i $(STACK_IP_ADDRESS),

.PHONY: create-stack
create-stack: provision-stack setup-stack

.PHONY: destroy-stack
destroy-stack:
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars="$(LABELS_STRING) nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeUser=$(STACK_IMAGE)" \
		--extra-vars="runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
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

.PHONY: provision-node
provision-node: .runID
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	@:$(call check_defined, STACK_IP_ADDRESS, IP address of the stack not defined)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) stackRunner=$(STACK_IP_ADDRESS) nodeLabel=$(NODE_LABEL) nodeImage=$(NODE_IMAGE) nodeInstanceType=$(NODE_INSTANCE_TYPE) nodeUser=$(NODE_USER)" \
		--extra-vars "runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ suite=$(SUITE) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t provision-node

.PHONY: setup-node
setup-node: .runID
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	@:$(call check_defined, STACK_IP_ADDRESS, IP address of the stack not defined)
	@:$(call check_defined, NODE_IP_ADDRESS, IP address of the test node not defined)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) stackRunner=$(STACK_IP_ADDRESS) nodeLabel=$(NODE_LABEL) nodeImage=$(NODE_IMAGE) nodeInstanceType=$(NODE_INSTANCE_TYPE) nodeUser=$(NODE_USER)" \
		--extra-vars "runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ suite=$(SUITE) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t setup-node \
		-i $(NODE_IP_ADDRESS),

.PHONY: create-node
create-node: provision-node setup-node

.PHONY: destroy-node
destroy-node:
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars="$(LABELS_STRING) nodeLabel=$(NODE_LABEL) nodeImage=$(NODE_IMAGE) nodeUser=$(NODE_USER)" \
		--extra-vars="runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t destroy
	rm -fr $(PROJECT_DIR)/.ci/.node-host-ip

.PHONY: fetch-test-reports
fetch-test-reports: .runID
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	@:$(call check_defined, NODE_IP_ADDRESS, IP address of the test node not defined)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) nodeLabel=$(NODE_LABEL) nodeImage=$(NODE_IMAGE) nodeInstanceType=$(NODE_INSTANCE_TYPE) nodeUser=$(NODE_USER)" \
		--extra-vars "nodeIP=$(NODE_IP_ADDRESS)" \
		--extra-vars "runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ suite=$(SUITE) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t fetch-reports \
		-i $(NODE_IP_ADDRESS),

.PHONY: ssh-node
ssh-node:
	ssh -i $(SSH_KEY) $(NODE_USER)@$(NODE_IP_ADDRESS)

.PHONY: show-node
show-node:
	@echo "Node IP            : $(NODE_IP_ADDRESS)"
	@echo "Node type          : $(NODE_LABEL)"
	@echo "Node instance type : $(NODE_INSTANCE_TYPE)"
	@echo "Node SSH user      : $(NODE_USER)"

.PHONY: destroy-elastic-stack
destroy-elastic-stack:
	ssh -i $(SSH_KEY) $(STACK_USER)@$(STACK_IP_ADDRESS) 'sudo docker-compose -f /root/.op/compose/profiles/fleet/docker-compose.yml down --remove-orphans'

.PHONY: recreate-fleet-server
recreate-fleet-server:
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	PROVIDER="remote" SUITE="$(SUITE)" TAGS="non-existent-tag" \
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeInstanceType=$(STACK_INSTANCE_TYPE) nodeUser=$(STACK_USER)" \
		--extra-vars "runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t scripts \
		-i $(STACK_IP_ADDRESS),
	ssh -i $(SSH_KEY) $(STACK_USER)@$(STACK_IP_ADDRESS) 'sudo bash /home/$(STACK_USER)/e2e-testing/.ci/scripts/functional-test.sh'

.PHONY: run-tests
run-tests:
	@:$(call check_defined, RUN_ID, You need to have an unique RUN_ID. To create it please run 'make .runID' goal)
	PROVIDER="$(PROVIDER)" SUITE="$(SUITE)" TAGS="$(TAGS)" REPORT_PREFIX="$(SUITE)_$(NODE_LABEL)_$(TAGS)" \
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "$(LABELS_STRING) stackRunner=$(STACK_IP_ADDRESS) nodeLabel=$(NODE_LABEL) nodeImage=$(NODE_IMAGE) nodeInstanceType=$(NODE_INSTANCE_TYPE) nodeUser=$(NODE_USER)" \
		--extra-vars "runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ suite=$(SUITE) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t scripts \
		-i $(NODE_IP_ADDRESS),
	ssh -i $(SSH_KEY) $(NODE_USER)@$(NODE_IP_ADDRESS) 'source /home/$(NODE_USER)/e2e-testing/.env; sudo -preserve-env bash /home/$(NODE_USER)/e2e-testing/.ci/scripts/functional-test.sh "$(TAGS)"'
