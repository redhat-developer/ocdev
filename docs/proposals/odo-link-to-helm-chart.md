# odo link to helm chart

## Motivation
Helm is probably the most widely used tool to deploy standardized services (databases, caches, proxies etc..) on Kubernets.
Odo users should be able easily consume services deployed using Helm.


## User stories
### User Story 1
As a developer I want to be able to create a link (`odo link <helm-chart>`) between my component and a database that I’ve deployed using Helm so I can connect to a database from my application.

### User Story 2
As a developer consuming services deployed using Helm I want to be able to list what I can consume (link to).

## Design overview
###  Changes in `odo link` 
To cover [User Story 1](#user-story-1) `odo link` command needs to be able to generate ServiceBinding that can bind odo component to deployed Helm Chart.


#### Notes/Challenges/Questions/Problems:
- What should be target (`sepc.services`) in `ServiceBinding` CR? How can SBO detect correct binding information?
- When I used `Deployment` or `StatefullSet` in `spec.services` SBO did not detected any binding information (`Secret` was empty). Reason for this is that the Charts that I tested this with don't specify OwnersReference.
    - What about binding directly to a `Secret` that  is usually created by Helm?
        - How to find correct secret?
        - Does this mean that every "connectable" Helm Chart will require `Secret`?
        - Does `Secret` includes hostname? - **NO**
            - It doesn't, usually just passwords are stored  there.
                - this is case of https://artifacthub.io/packages/helm/t3n/mysql
                - also https://artifacthub.io/packages/helm/bitnami/mysql (https://github.com/bitnami/charts/blob/master/bitnami/mysql/templates/secrets.yaml)
                - https://artifacthub.io/packages/helm/bitnami/mongodb (https://github.com/bitnami/charts/blob/master/bitnami/mongodb/templates/secrets.yaml)
            - How to add user information to binding?
            - How to add hostname and port information to binding?

One approach that I tried was creating SB with all `Secrets` and `Services` that were created by given Helm Chart.
This should theoretically be enough to provide application with hostname and  passwords.

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: ServiceBinding
metadata:
  name: mysql-binding
  namespace: tkral-test
spec:
  detectBindingResources: true
  application:
    group: apps
    version: v1
    resource: deployments
    name: python-mysql

  services:
    - group: ""
      version: v1
      kind: Service
      name: my-mysql
    - group: ""
      version: v1
      kind: Secret
      name: my-mysql

```


**This currently doesn't work as SBO doesn't support direct binding. https://github.com/redhat-developer/service-binding-operator/issues/872#issuecomment-779363582**


### Changes in  `odo service`
To cover [User Story 2](user-story-2) `odo service list` command  needs to list all deployed Helm Charts.
The Name field in the list output should uniquely identify service (Helm Chart) so users can use that name in `odo link` command.
```bash
odo link <service-name>
```

One approach could be to list `Service`s defined in Helm charts
```bash
kubectl get svc -l app.kubernetes.io/managed-by=Helm -o jsonpath="Service.name | Helm.release-name{'\n'}{range .items[*]}{.metadata.name}  |  {.metadata.annotations.meta\.helm\.sh\/release-name}{'\n'}{end}"
```
If odo will use `Secrets` as a binding information for SBO, we should probably use `Secrets` instead of `Services` for listing


