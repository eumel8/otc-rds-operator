# otc-rds-operator

Kubernetes Operator for OTC RDS

[![Project Status: WIP – Initial development is in progress, but there has not yet been a stable, usable release suitable for the public.](https://www.repostatus.org/badges/latest/wip.svg)](https://www.repostatus.org/#wip)

`master` branch might be broken, look at the [release page](https://github.com/eumel8/otc-rds-operator/releases) 
to use a functional version.

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
* Backup restore PITR
* Log handling
* (User handling)

## Versioning 

|Rds      | Job | CronJob | Lease | Kubernetes |
|---------|-----|---------|-------|------------|
|v1alpha1 | v1  | v1      | v1    | v1.21.x    |

## Installation

Look at the `otc` section in the [chart/values.yaml](chart.values.yaml) to provide credentials.
Best case to generate your own values.yaml and install the app:

```bash
helm -n rdsoperator upgrade -i rdsoperator -f values.yaml chart --create-namespace
```

In a minimal set and if you use `eu-de` region, only `domain_name`, `username`, and `password` are required:

```bash
helm -n rdsoperator upgrade -i rdsoperator --set otc.domain_name=<OTC-EU-DE-000000000010000000001> --set otc.username=<user> --set otc.password=<password> chart --create-namespace
```

## Custom Resource Definitions (CRDs)

In [manifests/crds/rds.yml](manifests/crds/rds.yml) the used CRD is located.

```yaml
Group: otc.mcsps.de
Kind: Rds
```
 
## Usage

Install a MySQL Single Instance:

```bash
kubectl apply -f ./manifests/examples/my-rds-single.yml
```

Install a MySQL HA Instance:

```bash
kubectl apply -f ./manifests/examples/my-rds-ha.yml
```

hint: The root password will be deleted in the resource spec after creating RDS.

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

Edit the status of the RDS instance and set `reboot: true`. Instance will restart immediatelly.

Example:

```yaml
status:
  id: 59ce9eca5ef543e2a643fea393b2cfbcin01
  ip: 192.168.0.71
  logs: false
  reboot: true
  status: ACTIVE
```

### Backup Restore PITR

Set `backuprestoretime` to a valid date. RDS will recover immediatelly:

```yaml
spec:
  backuprestoretime: "2022-04-20T22:08:41+00:00"
```

### Fetch RDS Logs

Edit the status of the RDS instance and set `logs: true`. A Job will started in the same
namespac to collect and show logs.

Example:

```yaml
status:
  id: 59ce9eca5ef543e2a643fea393b2cfbcin01
  ip: 192.168.0.71
  logs: true
  reboot: false
  status: ACTIVE
```

Query Job for logs:

```bash
$ kubectl -n rds2 logs -ljob-name=my-rds-ha -c errorlog
    "time": "2022-04-26T15:23:08Z",
    "level": "WARNING",
    "content": "[MY-011070] [Server] \u0026#39;Disabling symbolic links using --skip-symbolic-links (or equivalent) is the default. Consider not using this option as it\u0026#39; is deprecated and will be removed in a future release."
  },
]
```

### Import/Export

#### Import

If you want to migrate an unmanaged RDS instance into the operator, create a resource
and apply to the cluster. Operator care about existing instances and do not create it twice

1. Lookup for the unmanaged instance

```bash
openstack rds instance list

+--------------------------------------+-------------------+----------------+-------------------+--------+------------------------+--------+------+--------+
| ID                                   | Name              | Datastore Type | Datastore Version | Status | Flavor_ref             | Type   | Size | Region |
+--------------------------------------+-------------------+----------------+-------------------+--------+------------------------+--------+------+--------+
| 6780320f1c1249d6922e0c4573a697a0in01 | my-rds-ha2        | MySQL          | 8.0               | ACTIVE | rds.mysql.c2.medium.ha | Ha     |  100 | eu-de  |
```

2. Create manifest:

```yaml
apiVersion: otc.mcsps.de/v1alpha1
kind: Rds
metadata:
  name: my-rds-ha2
spec:
  datastoretype: "MySQL"
  datastoreversion: "8.0"
  volumetype: "COMMON"
  volumesize: 100
  hamode: "Ha"
  hareplicationmode: "semisync"
  port: "3306"
  password: "A12345678+"
  backupstarttime: "01:00-02:00"
  backupkeepdays: 10
  flavorref: "rds.mysql.c2.medium.ha"
  region: "eu-de"
  availabilityzone: "eu-de-01,eu-de-02"
  vpc: "golang"
  subnet: "golang"
  securitygroup: "golang"
```

3. Apply to the target namespace in the cluster. No you can operate all supported usecases

#### Export

Remove the `Id` from the Rds resource

```yaml
Status:
  Id:      8a725142698e4d0a9ed606d97905a77din01
```

After that the instance in OTC is unmanaged and you can safely delete Rds in Kubernetes Cluster.

## Developement

Using `make` in the doc root:

```bash
           fmt: Format
           run: Run controller
           vet: Vet
          deps: Optimize dependencies
          help: Show targets documentation
          test: Run tests
         build: Build binary
         clean: Clean build files
         cover: Run tests and generate coverage
         mocks: Generate mocks
        vendor: Vendor dependencies
      generate: Generate code
    test-clean: Clean test cache
```

## Credits

Frank Kloeker f.kloeker@telekom.de

Life is for sharing. If you have an issue with the code or want to improve it, feel free to open an issue or an pull request.

The Operator is inspired by [@mmontes11](https://github.com/mmontes11/echoperator), a good place
to learn fundamental things about K8S API.
