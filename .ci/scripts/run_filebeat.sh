#!/usr/bin/env bash
set -eu
OUTPUT_DIR=${OUTPUT_DIR:-'/tmp/filebeat'}
OUTPUT_FILE=${OUTPUT_FILE:-'docker'}
CONFIG_PATH=${CONFIG_PATH:-'/tmp/filebeat.yml'}
DOCKER_IMAGE=${DOCKER_IMAGE:-'docker.elastic.co/beats/filebeat:8.5.3'}

echo "OUTPUT_DIR=${OUTPUT_DIR}"
echo "OUTPUT_FILE=${OUTPUT_FILE}"
echo "CONFIG_PATH=${CONFIG_PATH}"
echo "DOCKER_IMAGE=${DOCKER_IMAGE}"

for c in $(docker ps --filter label="name=filebeat" -q)
do
  docker kill "${c}"
done

mkdir -p "${OUTPUT_DIR}"

cat <<EOF > "${CONFIG_PATH}"
---
filebeat.autodiscover:
  providers:
    - type: docker
      condition:
        not:
          contains:
            container.image: ${DOCKER_IMAGE}
      templates:
        - config:
          - type: container
            paths:
              - /var/lib/docker/containers/\${data.docker.container.id}/*.log
processors:
  - add_host_metadata: ~
  - add_cloud_metadata: ~
  - add_docker_metadata: ~
  - add_kubernetes_metadata: ~

filebeat.inputs:
- type: filestream
  id: elastic-agent-logs
  paths:
    - /var/lib/elastic-agent/data/elastic-agent-*/logs/*.log*
    - /Library/Elastic/Agent/data/elastic-agent-*/logs/*.log*

output.file:
  path: "/output"
  filename: ${OUTPUT_FILE}
  permissions: 0644
  codec.format:
    string: '{"image": "%{[container.image.name]}", "message": %{[message]}}'
EOF

echo "INFO: Run filebeat"
docker run \
  --detach \
  -v "${OUTPUT_DIR}:/output" \
  -v "${CONFIG_PATH}:/usr/share/filebeat/filebeat.yml" \
  -u 0:0 \
  -v /var/lib/docker/containers:/var/lib/docker/containers \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e OUTPUT_FILE="${OUTPUT_FILE}" \
  -p 5066:5066 \
  "${DOCKER_IMAGE}" \
    --strict.perms=false \
    -environment container \
    -E http.enabled=true > filebeat_docker_id

ID=$(docker ps --filter label="name=filebeat" -q)
URL=${2:-"http://localhost:5066/stats?pretty"}

echo "INFO: print existing docker context"
docker ps -a || true

sleep 10

echo "INFO: wait for the docker container to be available"
N=0
until docker exec "${ID}" curl -sSfI --retry 10 --retry-delay 5 --max-time 5 "${URL}"
do
  sleep 5
  if [ "${N}" -gt 6 ]; then
    echo "ERROR: print docker inspect"
    docker inspect "${ID}"
    echo "ERROR: docker container is not available"
    docker logs "${ID}"
    break;
  fi
  N=$((N + 1))
done
