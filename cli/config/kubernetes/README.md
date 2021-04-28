# K8s deployment for Elasticsearch, Kibana, and Fleet-Server

## Requirements

- docker
- kind (>= 0.10.0)
- kubectl (>= 1.17)

## Deployment

```
kind create cluster
kubectl apply -k base
```

## Local Ingress

```
kubectl apply -k overlays/local
```

This will allow you to reach the Kibana endpoint at `http://localhost`
