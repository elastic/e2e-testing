# K8s deployment for Elasticsearch, Kibana, and Fleet-Server

## Requirements

- docker
- kind (>= 0.14.0)
- kubectl (>= 1.25.0)

## Deployment

```
kind create cluster
kubectl apply -k base
```

This will allow you to visit `http://localhost:5601` using username: `elastic` password: `changeme` to login

## Local Ingress (Optional)

If a local ingress is preferred, apply the following to setup nginx-ingress

```
kubectl apply -k overlays/local
```

This will allow you to reach the Kibana endpoint at `http://localhost`
