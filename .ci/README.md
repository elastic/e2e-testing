# Running a CI Deployment

## Prereqs

The following variables need to be exported:

- *RUN_ID*: This is a unique identifying ID for the current run. It can be an arbitrary name or something like this:

```
export RUN_ID=$(uuidgen|cut -d'-' -f1))
```

- *AWS_SECRET_ACCESS_KEY*: AWS secret access key
- *AWS_ACCESS_KEY_ID*: AWS access key id

Install python deps:

```
> python3 -mvenv venv
> venv/bin/pip3 install ansible requests google-auth boto3 boto
> venv/bin/ansible-galaxy install geerlingguy.docker
```

## Deploy stack with X number of agents

```
> venv/bin/ansible-playbook .ci/ansible/playbook.yml \
    --extra-vars "numAgents=1 runId=$RUN_ID workspace=$(pwd)/ sshPublicKey=~/.ssh/id_rsa.pub" \
    --ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
    -t provision-stack
```
