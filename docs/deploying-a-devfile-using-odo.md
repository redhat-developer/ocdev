---
# Page settings
layout: default
keywords:
comments: false

# Hero section
title: Deploying your first application using odo
description: Learn how to deploy an application using odo and Devfile

# Micro navigation
micro_nav: true

# Page navigation
page_nav:
    prev:
        content: Understanding odo
        url: '/docs/understanding-odo'
    next:
        content: Deploy a devfile on IBM Z & Power
        url: '/docs/deploying-a-devfile-using-odo-on-IBM-Z-and-Power'
---
# Introduction to devfile

What is a devfile?

A [devfile](https://redhat-developer.github.io/devfile/) is a portable file that describes your development environment. It allows reproducing a *portable* developmental environment without the need of reconfiguration.

With a devfile you can describe:

  - Development components such as container definition for build and application runtimes

  - A list of pre-defined commands that can be run

  - Projects to initially clone

odo takes this devfile and transforms it into a workspace of multiple containers running on Kubernetes or OpenShift.

Devfiles are YAML files with a defined [schema](https://devfile.github.io/devfile/_attachments/api-reference.html).

# odo and devfile

odo can now create components from devfiles as recorded in registries. odo automatically consults the [default registry](https://github.com/odo-devfiles/registry) but users can also add their own registries. Devfiles contribute new component types that users can pull to begin development immediately.

An example deployment scenario:

1.  `odo create` will consult the recorded devfile registries to offer the user a selection of available component types and pull down the associated `devfile.yaml` file

2.  `odo push` parses and then deploys the component in the following order:
    
    1.  Parses and validates the YAML file
    
    2.  Deploys the development environment to your Kubernetes or OpenShift cluster
    
    3.  Synchronizes your source code to the containers
    
    4.  Executes any prerequisite commands

# Deploying your first devfile

## Ingress Setup

You will need to provide an ingress domain name for the services you create with odo, which you will specify via the `--host` argument with `odo url create`.

An easy way to do this is to use the [nip.io](https://nip.io/) service to create host names mapping to the external IP of your ingress controller service.

In the commands below we assume you are using the [nip.io](https://nip.io/) service and [minikube](https://minikube.sigs.k8s.io/docs/), e.g.:

``` sh
  $ odo url create --host $(minikube ip).nip.io
```

### Minikube Cluster Ingress Setup

Enable the ingress addon in minikube.

``` sh
  $ minikube addons enable ingress
```

With Minikube running in a virtual machine, the ingress controller IP address is obtained via `minikube ip` (the value is `192.168.99.100` in the sample output shown below).

### OpenShift Cluster Ingress Setup

For [CodeReady Containers](https://developers.redhat.com/products/codeready-containers/overview) running in a virtual machine, for example, you can get it by `crc ip`. Note you might not need to use the ingress in OpenShift, where you can use routes instead.

### Ingress Notes

Of course there are other options but this approach avoids the need to edit the `/etc/hosts/` file with each created service URL.

Checkout this [document](https://kubernetes.io/docs/concepts/services-networking/ingress/) to know more about ingress.

## First Steps

  - Login to your cluster (unnecessary if you’ve used other standard methods, e.g. kubectl to establish the current context)
    
    ``` sh
      $ odo login -u developer -p developer
    ```

  - Create a project to keep your source code, tests, and libraries organized in a separate single unit.
    
    ``` sh
      $ odo project create myproject
       ✓  Project 'myproject' is ready for use
       ✓  New project created and now using project : myproject
    ```

# Listing all available devfile components

  - Before deploying your first component, have a look at what is available:
    
    ``` sh
      $ odo catalog list components
      Odo Devfile Components:
      NAME                 DESCRIPTION                            REGISTRY
      java-maven           Upstream Maven and OpenJDK 11          DefaultDevfileRegistry
      java-openliberty     Open Liberty microservice in Java      DefaultDevfileRegistry
      java-quarkus         Upstream Quarkus with Java+GraalVM     DefaultDevfileRegistry
      java-springboot      Spring Boot® using Java                DefaultDevfileRegistry
      nodejs               Stack with NodeJS 12                   DefaultDevfileRegistry
    
      Odo S2I Components:
      NAME        PROJECT       TAGS                                                                           SUPPORTED
      java        openshift     11,8,latest                                                                    YES
      dotnet      openshift     2.1,3.1,latest                                                                 NO
      golang      openshift     1.13.4-ubi7,1.13.4-ubi8,latest                                                 NO
      httpd       openshift     2.4-el7,2.4-el8,latest                                                         NO
      nginx       openshift     1.14-el7,1.14-el8,1.16-el7,1.16-el8,latest                                     NO
      nodejs      openshift     10-ubi7,10-ubi8,12-ubi7,12-ubi8,latest                                         NO
      perl        openshift     5.26-el7,5.26-ubi8,5.30-el7,latest                                             NO
      php         openshift     7.2-ubi7,7.2-ubi8,7.3-ubi7,7.3-ubi8,latest                                     NO
      python      openshift     2.7-ubi7,2.7-ubi8,3.6-ubi7,3.6-ubi8,3.8-ubi7,3.8-ubi8,latest                   NO
      ruby        openshift     2.5-ubi7,2.5-ubi8,2.6-ubi7,2.6-ubi8,2.7-ubi7,latest                            NO
      wildfly     openshift     10.0,10.1,11.0,12.0,13.0,14.0,15.0,16.0,17.0,18.0,19.0,20.0,8.1,9.0,latest     NO
    ```

In our example, we will be using `java-springboot` to deploy a sample [Springboot](https://spring.io/projects/spring-boot) component.

# Deploying a Java Spring Boot® component to a Kubernetes / OpenShift cluster

In this example we will be deploying an [example Spring Boot® component](https://github.com/odo-devfiles/springboot-ex) that uses [Maven](https://maven.apache.org/install.html) and Java 8 JDK.

1.  Download the example Spring Boot® component.
    
    ``` sh
     $ git clone https://github.com/odo-devfiles/springboot-ex
    ```
    
    Alternatively, you can pass in `--starter` to `odo create` to have odo download a project specified in the devfile.

2.  Change the current directory to the component directory:
    
    ``` sh
     $ cd <directory-name>
    ```

3.  Create a component configuration using the `java-springboot` component-type named `myspring`:
    
    ``` sh
       $ odo create java-springboot myspring
    
       Validation
        ✓  Checking devfile compatibility [195728ns]
        ✓  Creating a devfile component from registry: DefaultDevfileRegistry [170275ns]
        ✓  Validating devfile component [281940ns]
    
        Please use odo push command to create the component with source deployed
    ```

4.  List the contents of the directory to see the devfile and sample Java application source code:
    
    ``` sh
      $ ls
      README.md devfile.yaml    pom.xml     src
    ```

5.  Create a URL in order to access the deployed component:
    
    > **Note**
    > 
    > If deploying on OpenShift, you can skip this step and a Route will be created for you automatically. On Kubernetes, you need to pass ingress domain name via `--host` flag.
    
    ``` sh
     $ odo url create  --host $(minikube ip).nip.io
      ✓  URL myspring-8080.192.168.99.100.nip.io created for component: myspring
    
     To apply the URL configuration changes, please use odo push
    ```

6.  Push the component to the cluster:
    
    ``` sh
      $ odo push
    
      Validation
       ✓  Validating the devfile [81808ns]
    
      Creating Kubernetes resources for component myspring
       ✓  Waiting for component to start [5s]
    
      Applying URL changes
       ✓  URL myspring-8080: http://myspring-8080.192.168.99.100.nip.io created
    
      Syncing to component myspring
       ✓  Checking files for pushing [2ms]
       ✓  Syncing files to the component [1s]
    
      Executing devfile commands for component myspring
       ✓  Executing devbuild command "/artifacts/bin/build-container-full.sh" [1m]
       ✓  Executing devrun command "/artifacts/bin/start-server.sh" [2s]
    
      Pushing devfile component myspring
       ✓  Changes successfully pushed to component
    ```

7.  List the URLs of the component:
    
    ``` sh
     $ odo url list
     Found the following URLs for component myspring
     NAME              URL                                       PORT     SECURE
     myspring-8080     http://myspring-8080.192.168.99.100.nip.io     8080     false
    ```

8.  View your deployed application using the generated URL:
    
    ``` sh
      $ curl http://myspring-8080.$(minikube ip).nip.io
    ```

9.  To delete your deployed application:
    
    ``` sh
      $ odo delete
      ? Are you sure you want to delete the devfile component: myspring? Yes
       ✓  Deleting devfile component myspring [152ms]
       ✓  Successfully deleted component
    ```

# Deploying a Node.js® component to a Kubernetes / OpenShift cluster

In this example we will be deploying an [example Node.js® component](https://github.com/odo-devfiles/nodejs-ex) that uses [NPM](https://www.npmjs.com/).

1.  Download the example Node.js® component
    
    ``` sh
     $ git clone https://github.com/odo-devfiles/nodejs-ex
    ```

2.  Change the current directory to the component directory:
    
    ``` sh
     $ cd <directory-name>
    ```

3.  List the contents of the directory to confirm that the application is indeed a Node.js® application:
    
    ``` sh
     $ ls
     LICENSE  package.json  package-lock.json  README.md  server.js  test
    ```

4.  Create a component configuration using the `nodejs` component-type named `mynodejs`:
    
    ``` sh
     $ odo create nodejs mynodejs
    
     Validation
      ✓  Checking devfile compatibility [111738ns]
      ✓  Creating a devfile component from registry: DefaultDevfileRegistry [89567ns]
      ✓  Validating devfile component [186982ns]
    
     Please use odo push command to create the component with source deployed
    ```

5.  Create a URL in order to access the deployed component:
    
    > **Note**
    > 
    > If deploying on OpenShift, you can skip this step and a Route will be created for you automatically. On Kubernetes, you need to pass ingress domain name via `--host` flag.
    
    ``` sh
     $ odo url create --host $(minikube ip).nip.io
      ✓  URL mynodejs-8080.192.168.99.100.nip.io created for component: mynodejs
    
     To apply the URL configuration changes, please use odo push
    ```

6.  Push the component to the cluster:
    
    ``` sh
      $ odo push
    
      Validation
       ✓  Validating the devfile [89380ns]
    
      Creating Kubernetes resources for component mynodejs
       ✓  Waiting for component to start [3s]
    
      Applying URL changes
       ✓  URL mynodejs-3000: http://mynodejs-3000.192.168.99.100.nip.io created
    
      Syncing to component mynodejs
       ✓  Checking files for pushing [2ms]
       ✓  Syncing files to the component [1s]
    
      Executing devfile commands for component mynodejs
       ✓  Executing devbuild command "npm install" [3s]
       ✓  Executing devrun command "nodemon app.js" [2s]
    
      Pushing devfile component mynodejs
       ✓  Changes successfully pushed to component
    ```

7.  List the URLs of the component:
    
    ``` sh
     $ odo url list
         Found the following URLs for component mynodejs
         NAME              URL                                            PORT     SECURE
         mynodejs-8080     http://mynodejs-8080.192.168.99.100.nip.io     8080     false
    ```

8.  View your deployed application using the generated URL:
    
    ``` sh
       $ curl http://mynodejs-8080.$(minikube ip).nip.io
    ```

9.  To delete your deployed application:
    
    ``` sh
       $ odo delete
       ? Are you sure you want to delete the devfile component: mynodejs? Yes
        ✓  Deleting devfile component mynodejs [139ms]
        ✓  Successfully deleted component
    ```

# Deploying a Quarkus Application to a Kubernetes / OpenShift cluster

In this example we will be deploying a [Quarkus component](https://github.com/odo-devfiles/quarkus-ex) that uses GraalVM and JDK1.8+.

1.  Download the example Quarkus component
    
    ``` sh
     $ git clone https://github.com/odo-devfiles/quarkus-ex && cd quarkus-ex
    ```

2.  Create a Quarkus odo component
    
    ``` sh
       $ odo create java-quarkus myquarkus
    
       Validation
        ✓  Checking devfile compatibility [195728ns]
        ✓  Creating a devfile component from registry: DefaultDevfileRegistry [170275ns]
        ✓  Validating devfile component [281940ns]
    
        Please use odo push command to create the component with source deployed
    ```

3.  Create a URL in order to access the deployed component:
    
    > **Note**
    > 
    > If deploying on OpenShift, you can skip this step and a Route will be created for you automatically. On Kubernetes, you need to pass ingress domain name via `--host` flag.
    
    ``` sh
     $ odo url create  --host $(minikube ip).nip.io
      ✓  URL myquarkus-8080.192.168.99.100.nip.io created for component: myquarkus
    
     To apply the URL configuration changes, please use odo push
    ```

4.  Push the component to the cluster:
    
    ``` sh
      $ odo push
    
    Validation
     ✓  Validating the devfile [44008ns]
    
    Creating Kubernetes resources for component myquarkus
     ✓  Waiting for component to start [10s]
    
    Applying URL changes
     ✓  URLs are synced with the cluster, no changes are required.
    
    Syncing to component myquarkus
     ✓  Checking files for pushing [951138ns]
     ✓  Syncing files to the component [204ms]
    
    Executing devfile commands for component myquarkus
     ✓  Executing init-compile command "mvn compile" [3m]
     ✓  Executing dev-run command "mvn quarkus:dev" [1s]
    
    Pushing devfile component myquarkus
     ✓  Changes successfully pushed to component
    ```

5.  View your deployed application in a browser using the generated url
    
    ``` sh
     $ odo url list
     Found the following URLs for component myspring
     NAME              URL                                              PORT     SECURE
     myquarkus-8080     http://myquarkus-8080.192.168.99.100.nip.io     8080     false
    ```

You can now continue developing your application. Just run `odo push` and refresh your browser to view the latest changes.

You can also run `odo watch` to watch changes in the source code. Just refreshing the browser will render the source code changes.

Run `odo delete` to delete the application from cluster.

1.  To delete your deployed application:
    
    ``` sh
       $ odo delete
       ? Are you sure you want to delete the devfile component: java-springboot? Yes
        ✓  Deleting devfile component java-springboot [139ms]
        ✓  Successfully deleted component
    ```

# Deploying an Open Liberty Application to an OpenShift / Kubernetes cluster

In this example we will be deploying a [Open Liberty component](https://github.com/OpenLiberty/application-stack-intro) that uses Open Liberty and OpenJ9.

1.  Download the example Open Liberty component
    
    ``` sh
     $ git clone https://github.com/OpenLiberty/application-stack-intro.git && cd application-stack-intro
    ```

2.  Create an Open Liberty odo component
    
    ``` sh
       $ odo create myopenliberty
    
       Validation
        ✓  Creating a devfile component from devfile path: .../application-stack-intro/devfile.yaml [253220ns]
        ✓  Validating devfile component [263521ns]
    
       Please use `odo push` command to create the component with source deployed
    ```

3.  Create a URL in order to access the deployed component:
    
    > **Note**
    > 
    > If deploying on OpenShift, you can skip this step and a Route will be created for you automatically. On Kubernetes, you need to pass ingress domain name via `--host` flag.
    
    ``` sh
     $ odo url create --host $(minikube ip).nip.io
      ✓  URL myopenliberty-9080 created for component: myopenliberty
    
     To apply the URL configuration changes, please use odo push
    ```

4.  Push the component to the cluster:
    
    ``` sh
      $ odo push
    
    Validation
     ✓  Validating the devfile [72932ns]
    
    Creating Kubernetes resources for component myopenliberty
     ✓  Waiting for component to start [23s]
    
    Syncing to component myopenliberty
     ✓  Checking files for pushing [4ms]
     ✓  Syncing files to the component [4s]
    
    Executing devfile commands for component myopenliberty
     ✓  Executing build command "if [ -e /projects/.disable-bld-cmd ]; then echo \"found the disable file\" && echo \"devBuild command will not run\" && exit 0; else echo \"will run the devBuild command\" && mkdir -p /projects/target/liberty && if [ ! -d /projects/target/liberty/wlp ]; then echo \"...moving liberty\"; mv /opt/ol/wlp /projects/target/liberty; touch ./.liberty-mv; elif [[ -d /projects/target/liberty/wlp && ! -e /projects/.liberty-mv ]]; then echo \"STACK WARNING - LIBERTY RUNTIME WAS LOADED FROM HOST\"; fi && mvn -Dliberty.runtime.version=20.0.0.10 package && touch ./.disable-bld-cmd; fi" [9s]
     ✓  Executing run command "mvn -Dliberty.runtime.version=20.0.0.10 -Ddebug=false -DhotTests=true -DcompileWait=3 liberty:dev", if not running [2s]
    
    Pushing devfile component myopenliberty
     ✓  Changes successfully pushed to component
    ```

5.  List the URLs of the component
    
    ``` sh
     $ odo url list
      Found the following URLs for component myopenliberty
      NAME                STATE      URL                                                 PORT     SECURE
      myopenliberty-9     Pushed     http://myopenliberty-9.192.168.99.100.nip.io        9080     false
    ```

6.  View your deployed application using the generated URL (this example shows an ingress hostname URL, while an OpenShift route would look a bit different):
    
    ``` sh
     $ curl http://myopenliberty-9.$(minikube ip).nip.io/api/resource
    ```

7.  Have odo watch for changes in the source code:
    
    ``` sh
     $ odo watch
    ```

You can now continue developing your application. Refreshing the browser or hitting the endpoint again will render the source code changes.

You can also trigger `odo watch` with custom devfile build, run and debug commands.

``` sh
$ odo watch --build-command="mybuild" --run-command="myrun" --debug-command="mydebug"
----

Run `odo delete` to delete the application from cluster.

. To delete your deployed application:
+
[source,sh]
----
   $ odo delete
   Are you sure you want to delete the devfile component: myopenliberty? Yes

   Gathering information for component myopenliberty
    ✓  Checking status for component [99ms]

   Deleting component myopenliberty
    ✓  Deleting Kubernetes resources for component [107ms]
    ✓  Successfully deleted component

----
```
