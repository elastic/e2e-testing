# Running a CI Deployment

## Prereqs

The following variables need to be exported:

- *RUN_ID*: This is a unique identifying ID for the current run. It can be an arbitrary name or something like this:

```
export RUN_ID=$(uuidgen|cut -d'-' -f1)
```

- *AWS_SECRET_ACCESS_KEY*: AWS secret access key
- *AWS_ACCESS_KEY_ID*: AWS access key id

Install python deps:

```
> python3 -mvenv venv
> venv/bin/pip3 install ansible requests boto3 boto
> venv/bin/ansible-galaxy install -r .ci/ansible/requirements.yml
```

## Deploy stack

```
> venv/bin/ansible-playbook .ci/ansible/playbook.yml \
    --private-key="$HOME/.ssh/id_rsa" \
    --extra-vars "nodeLabel=stack nodeImage=ami-0d90bed76900e679a nodeInstanceType=c5.4xlarge" \
    --extra-vars "runId=$RUN_ID workspace=$HOME/Projects/e2e-testing/ sshPublicKey=$HOME/.ssh/id_rsa.pub" \
    --ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
    -t provision-stack
```

Make note of the IP address displayed in the ansible summary.

## Setup stack

```
> venv/bin/ansible-playbook .ci/ansible/playbook.yml \
    --private-key="$HOME/.ssh/id_rsa" \
    --extra-vars "nodeLabel=stack nodeImage=ami-0d90bed76900e679a nodeInstanceType=c5.4xlarge" \
    --extra-vars "runId=$RUN_ID workspace=$HOME/Projects/e2e-testing/ sshPublicKey=$HOME/.ssh/id_rsa.pub" \
    --ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
    -t setup-stack \
    -i <ip address above>,
```

**Note**: The comma at the end of the ip address is required.

## Deploy test node

```
> venv/bin/ansible-playbook .ci/ansible/playbook.yml \
    --private-key="$HOME/.ssh/id_rsa" \
    --extra-vars "stackRunner=<ip address from above> nodeLabel=fleet_amd64 nodeImage=ami-0d90bed76900e679a nodeInstanceType=c5.4xlarge" \
    --extra-vars "runId=$RUN_ID workspace=$HOME/Projects/e2e-testing/ sshPublicKey=$HOME/.ssh/id_rsa.pub" \
    --ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
    -t setup-stack
```

Make note of the ip address displayed in the ansible summary.

## Setup test node

```
> venv/bin/ansible-playbook .ci/ansible/playbook.yml \
    --private-key="$HOME/.ssh/id_rsa" \
    --extra-vars "stackRunner=<ip address from above> nodeLabel=fleet_amd64 nodeImage=ami-0d90bed76900e679a nodeInstanceType=c5.4xlarge" \
    --extra-vars "runId=$RUN_ID workspace=$HOME/Projects/e2e-testing/ sshPublicKey=$HOME/.ssh/id_rsa.pub" \
    --ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
    -t setup-node \
    -i <ip address of node from above>,
```

**Note**: The comma at the end of the ip address is required.

## Run a test suite

```
> ssh -i $HOME/.ssh/id_rsa admin@<node ip address>
node> sudo bash e2e-testing/.ci/scripts/functional-test.sh "fleet_mode_agent"
```
