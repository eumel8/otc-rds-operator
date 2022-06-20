# otc-rds-operator

Kubernetes Operator for OTC RDS

[![Project Status: Active](https://www.repostatus.org/badges/latest/active.svg)](https://www.repostatus.org/#active)

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
* User/Schema handling
* Autopilot (automatically resource handling of mem, cpu, disc)

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

### Creating Database

Install a MySQL Single Instance:

```bash
kubectl apply -f ./manifests/examples/my-rds-single.yml
```

Install a MySQL HA Instance:

```bash
kubectl apply -f ./manifests/examples/my-rds-ha.yml
```

hint: The root password will be deleted in the resource spec after creating RDS.

### Creating User/Schema

To create database schema, add the names as a list in the RDS spec:

```yaml
spec:
  databases:
  - project1
  - project2
```

To create user, add a user object with name, host, password, and privileges in the RDS spec:

```yaml
spec:
  users:
  - host: 10.9.3.%
    name: app1
    password: app1+Mond
    privileges:
    - GRANT ALL ON project1.* TO 'app1'@'10.9.3.%'
```

hint: `host` is the allowed host/network to connect to the database for this user. Must equal
with the host defintion in `privileges` list.

Databases/Users/Privileges will created but not deleted within the Controller. If the database or
user is missing, the Controller will recreate it. Privileges will not adjust if the user exists.

### Usage of the Database

To use the database in an app deployment beside credentials the database hostname is required.
This can be fetched by command

```bash
kubectl -n rds1 get rds my-rds-single -o go-template='{{.status.ip}}'
10.9.3.181
```

OTC RDS Operator provides additionally a [Headless Service](https://kubernetes.io/docs/concepts/services-networking/service/#without-selectors). Application can use the service name, where the endpoint IP updates automatically
by the Operator.

```bash
kubectl -n rds1 get service my-rds-single
NAME            TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)    AGE
my-rds-single   ClusterIP   None         <none>        3306/TCP   9m17s
kubectl -n rds1 get ep my-rds-single
NAME            ENDPOINTS         AGE
my-rds-single   10.9.3.181:3306   9m22s
```

```bash
mysql -hmy-rds-single -uroot -p+
Welcome to the MariaDB monitor.  Commands end with ; or \g.
Your MySQL connection id is 397
Server version: 8.0.21-5 MySQL Community Server - (GPL)

Copyright (c) 2000, 2018, Oracle, MariaDB Corporation Ab and others.

Type 'help;' or '\h' for help. Type '\c' to clear the current input statement.

MySQL [(none)]>
```

Other solutions might be ExternalName-type services, which works only on hostnames,
not ip-addresses, because Cluster-DNS will generate a CNAME for this entry. This
requires a DNS name for the database, which can be realized by OTC DNS service. In
any case, the Kubernetes cluster needs a direct network connection to the database,
so it's installed in the same VPC/Subnet or it's available via VPC-Peering.

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
  autopilot: false
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
  autopilot: false
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

### Autopilot

Autopilot is level 5 approach of Kubernetes Operator. This means, the Operator care about health
status based on metrics cpu, memory, and disc size.
Autopilot will create 3 alert rules on OTC and a Simple Messaging Services for notification handling.
OTC fires a HTTPS webhook to a target service. For this it's required to setup Ingress, where the
Operator can receive alert notification and take action into account. Disc size will increase +10GB,
for cpu/memory alert the next bigger flavor will search and scale up the instance automatically,
while the cpu/mem metric is lower then 90%.
CloudEye will observe the load every 5 minutes and take action after 3 measurements (15 min), so
it takes some time before scale up the flavor.
For the moment there is no downsizing of flavors, and downsizing disc is always impossible.

Example:

```yaml
spec:
  endpoint: https://rdsoperator.example.com/
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

### Unforseen deleting

We identified two use cases of unforseen deleting of RDS instances:

* the Cluster Admin deletes the CRD, which caused the delete of all RDS instances in the Cluster
* the Cluster User deletes the RDS resource in the cluster which deletes of course the RDS instance in OTC

For both use cases we make a manual backup before delete the instance, which will be also not deleted
by the OTC delete cascade. You can restore the backup on the same instance name and import the RDS resource
into the cluster.

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

## Contribution

cncf.io/contributor-strategy

## Credits

Frank Kloeker f.kloeker@telekom.de

Life is for sharing. If you have an issue with the code or want to improve it, feel free to open an issue or an pull request.

The Operator is inspired by [@mmontes11](https://github.com/mmontes11/echoperator), a good place
to learn fundamental things about K8S API.
