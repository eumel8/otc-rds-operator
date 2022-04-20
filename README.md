# otc-rds-operator

Kubernetes operator for OTC RDS

## Prerequisites

* Kubernetes Cluster 1.21+ with cluster-admin permissions
* Existing OTC tenant with RDS Administrator permissions
* Existing OTC resources VPC,Subnet, SecurityGroup

## Features

* Create OTC RDS Instance
* Delete OTC RDS Instance
* Giving a status of the backend RDS (id,ip-address,state)
* Resize Flavor


## Versioning 

|Rds      | Job | CronJob | Lease | Kubernetes |
|---------|-----|---------|-------|------------|
|v1alpha1 | v1  | v1      | v1    | v1.21.x    |

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

@mmontes11 https://github.com/mmontes11/echoperator
