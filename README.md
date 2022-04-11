# otc-rds-operator

Kubernetes operator for OTC RDS

## Features

* Create OTC RDS Instance


## Versioning 

|Rds|Job|CronJob|Lease|Kubernetes|
|----|-------------|---|-------|-----|----------|
|v1alpha1|v1|v1|v1|v1.21.x|

## Installation

```bash
helm -n rdsoperator upgrade -i rdsoperator chart --create-namespace
```

## Custom Resource Definitions (CRDs)

## Usage

```bash
kubectl apply -f ./manifests/examples/my-rds.yml
```

## Credits
