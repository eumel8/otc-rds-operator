# Testing steps

```bash
# create namespaces
kubectl create namespace rds1
kubectl create namespace rds2
# create single/ha instances
kubectl -n rds1 apply -f manifests/examples/my-rds-single.yml
kubectl -n rds2 apply -f manifests/examples/my-rds-ha.yml
# describe resource
kubectl -n rds1 describe rds my-rds-single
kubectl -n rds2 describe rds my-rds-ha
# reboot instance
kubectl -n rds1 patch rds my-rds-single --type=json -p='[{"op": "replace", "path": "/status/reboot", "value": true}]'
kubectl -n rds1 describe rds my-rds-single
# Status:
#   Id:      8a725142698e4d0a9ed606d97905a77din01
#   Ip:      192.168.0.229
#   Reboot:  false
#   Status: REBOOTING
# Events:
#   Type    Reason  Age                From              Message
#   ----    ------  ----               ----              -------
#   Normal  Update  42m (x3 over 92m)  otc-rds-operator  This instance is rebooting.

# scale instance
kubectl -n rds patch rds my-rds-single --type=json -p='[{"op": "replace", "path": "/spec/flavorref", "value": "rds.mysql.c2.large"}]'
kubectl -n rds describe rds my-rds-single
# Status:
#   Id:      8a725142698e4d0a9ed606d97905a77din01
#   Ip:      192.168.0.229
#   Reboot:  false
#   Status: MODIFYING
# Events:
#   Type    Reason  Age                From              Message
#   ----    ------  ----               ----              -------
#   Normal  Update  7m21s              otc-rds-operator  This instance is scaling to rds.mysql.c2.medium

# status:
#   id: 8a725142698e4d0a9ed606d97905a77din01
#   ip: 192.168.0.229
#   reboot: false
#   status: MODIFYING
# !! flavors can be sold out in various AZ !!

# enlarge instance
kubectl -n rds patch rds my-rds-single --type=json -p='[{"op": "replace", "path": "/spec/volumesize", "value": 50}]'
kubectl -n rds describe rds my-rds-single
# Status:
#   Id:      8a725142698e4d0a9ed606d97905a77din01
#   Ip:      192.168.0.229
#   Reboot:  false
#   Status: MODIFYING
# Events:
#   Type    Reason  Age                From              Message
#   ----    ------  ----               ----              -------
#   Normal  Update  64s                otc-rds-operator  This instance is enlarging to 50

# restore backup
kubectl -n rds patch rds my-rds-single --type=json -p='[{"op": "replace", "path": "/spec/backuprestoretime", "value": "2022-04-25T22:08:41+00:00"}]
kubectl -n rds describe rds my-rds-single
Status:
  Id:      8a725142698e4d0a9ed606d97905a77din01
  Ip:      192.168.0.229
  Reboot:  false
  Status:  RESTORING
Events:
  Type    Reason  Age                From              Message
  ----    ------  ----               ----              -------
  Normal  Update  13s                otc-rds-operator  This instance is restoring to 2022-04-25T22:08:41+00:00

# delete instance
kubectl -n rds delete rds my-rds-single
kubectl -n rds describe rds my-rds-single
# Error from server (NotFound): rdss.otc.mcsps.de "my-rds-single" not found

# get instance logs
kubectl -n rds patch rds my-rds-single --type=json -p='[{"op": "replace", "path": "/status/logs", "value": "true"}]'
kubectl -n rds describe rds my-rds-single

# set autopilot
kubectl -n rds patch rds my-rds-single --type=json -p='[{"op": "replace", "path": "/spec/endpoint", "value": "https://rdsoperator.example.com/"}]'
kubectl -n rds describe rds my-rds-single
# generate some traffic for cpu/mem/disc scaling test 
# https://gist.github.com/eumel8/0226e518722fed3bc187eb89cca4d8a8

# remove autopilot
kubectl -n rds patch rds my-rds-single --type=json -p='[{"op": "replace", "path": "/status/autopilot", "value": "false"}]'
kubectl -n rds describe rds my-rds-single

# additional checkin background if cloud credentials available
openstack rds instance list
```
