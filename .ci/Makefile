PROJECT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/..
VENV_DIR=$(PROJECT_DIR)/.venv
RUN_ID_FILE=$(PROJECT_DIR)/.ci/.runID
RUN_ID:=$(shell cat $(RUN_ID_FILE))

SSH_KEY ?= $(HOME)/.ssh/id_rsa
SSH_KEY_PUBLIC = $(SSH_KEY).pub

# Debian 10 AMD (see .e2e-platforms.yaml)
STACK_IMAGE=ami-0d90bed76900e679a
STACK_INSTANCE_TYPE=c5.4xlarge
STACK_USER=admin
STACK_IP_ADDRESS ?= $(shell cat $(PROJECT_DIR)/.ci/.stack-host-ip)

# Debian 10 AMD (see .e2e-platforms.yaml)
NODE_IMAGE ?= ami-0d90bed76900e679a
NODE_INSTANCE_TYPE ?= c5.4xlarge
NODE_LABEL ?= debian_amd64
NODE_USER ?= admin
NODE_IP_ADDRESS ?= $(shell cat $(PROJECT_DIR)/.ci/.node-host-ip)

# Suite to run
SUITE ?= fleet

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
		--extra-vars "nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeInstanceType=$(STACK_INSTANCE_TYPE) nodeUser=$(STACK_USER)" \
		--extra-vars "runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t provision-stack

.PHONY: setup-stack
setup-stack: .runID
	@:$(call check_defined, RUN_ID, You need to an unique RUN_ID. To create it please run 'make .runID' goal)
	@:$(call check_defined, STACK_IP_ADDRESS, IP address of the stack)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeInstanceType=$(STACK_INSTANCE_TYPE) nodeUser=$(STACK_USER)" \
		--extra-vars "runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t setup-stack \
		-i $(STACK_IP_ADDRESS),

.PHONY: destroy-stack
destroy-stack:
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars="nodeLabel=stack nodeImage=$(STACK_IMAGE) nodeUser=$(STACK_IMAGE)" \
		--extra-vars="runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t destroy
	rm -fr $(PROJECT_DIR)/.ci/stack-host-ip

.PHONY: provision-node
provision-node: .runID
	@:$(call check_defined, RUN_ID, You need to an unique RUN_ID. To create it please run 'make .runID' goal)
	@:$(call check_defined, STACK_IP_ADDRESS, IP address of the stack)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "stackRunner=$(STACK_IP_ADDRESS) nodeLabel=$(NODE_LABEL) nodeImage=$(NODE_IMAGE) nodeInstanceType=$(NODE_INSTANCE_TYPE) nodeUser=$(NODE_USER)" \
		--extra-vars "runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ suite=$(SUITE) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t provision-node

.PHONY: setup-node
setup-node: .runID
	@:$(call check_defined, RUN_ID, You need to an unique RUN_ID. To create it please run 'make .runID' goal)
	@:$(call check_defined, STACK_IP_ADDRESS, IP address of the stack)
	@:$(call check_defined, NODE_IP_ADDRESS, IP address of the test node)
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars "stackRunner=$(STACK_IP_ADDRESS) nodeLabel=$(NODE_LABEL) nodeImage=$(NODE_IMAGE) nodeInstanceType=$(NODE_INSTANCE_TYPE) nodeUser=$(NODE_USER)" \
		--extra-vars "runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ suite=$(SUITE) sshPublicKey=$(SSH_KEY_PUBLIC)" \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t setup-node \
		-i $(NODE_IP_ADDRESS),

.PHONY: destroy-node
destroy-node:
	$(VENV_DIR)/bin/ansible-playbook $(PROJECT_DIR)/.ci/ansible/playbook.yml \
		--private-key="$(SSH_KEY)" \
		--extra-vars="nodeLabel=$(NODE_LABEL) nodeImage=$(NODE_IMAGE) nodeUser=$(NODE_USER)" \
		--extra-vars="runId=$(RUN_ID) workspace=$(PROJECT_DIR)/ sshPublicKey=$(SSH_KEY_PUBLIC) \
		--ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
		-t destroy
	rm -fr $(PROJECT_DIR)/.ci/node-host-ip
