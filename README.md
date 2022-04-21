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
* Enlarge Volume
* Restart Instance
* (Backup restore PITR)
* (Log handling)
* (User handling)


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

## Operation

### Resize Flavor

finding the available flavor for data type and version, and ha:

```bash
openstack rds flavor list mysql 8.0| grep ha
```

hint: only `ha` flavor are good for ha installations

### Enlarge Volume

Due the API spec change the volume size only upwards with a 10 GB step
and a minimum of 40 GB

### Restart Instance

Edit the status of the RDS instance and set `reboot: true`. Instance will restart immediatally.

Example:

```yaml
status:
  id: 59ce9eca5ef543e2a643fea393b2cfbcin01
  ip: 192.168.0.71
  reboot: false
  status: ACTIVE
```

## Credits

@mmontes11 https://github.com/mmontes11/echoperator
