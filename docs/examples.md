# Examples

Odo is compatible with any language or runtime listed within OpenShift's catalog of component types. In order to access the component over the web, you can create a URL using `odo url create`.

This can be found by using `odo catalog list components`.

Example:

```sh
NAME        PROJECT       TAGS
dotnet      openshift     2.0,latest
httpd       openshift     2.4,latest
java        openshift     8,latest
nginx       openshift     1.10,1.12,1.8,latest
nodejs      openshift     0.10,4,6,8,latest
perl        openshift     5.16,5.20,5.24,latest
php         openshift     5.5,5.6,7.0,7.1,latest
python      openshift     2.7,3.3,3.4,3.5,3.6,latest
ruby        openshift     2.0,2.2,2.3,2.4,latest
wildfly     openshift     10.0,10.1,8.1,9.0,latest
```


## Examples from Git repos

### httpd

Build and serve static content via httpd on CentOS 7. For more information about using this builder image, including OpenShift considerations, see https://github.com/sclorg/httpd-container/blob/master/2.4/root/usr/share/container-scripts/httpd/README.md.

```sh
  odo create httpd --git https://github.com/openshift/httpd-ex.git
```

### java 

Build and run fat JAR Java applications on CentOS 7. For more information about using this builder image, including OpenShift considerations, see https://github.com/fabric8io-images/s2i/blob/master/README.md.

```sh
  odo create java --git https://github.com/spring-projects/spring-petclinic.git
```

### nodejs

Build and run Node.js applications on CentOS 7. For more information about using this builder image, including OpenShift considerations, see https://github.com/sclorg/s2i-nodejs-container/blob/master/8/README.md.

```sh
  odo create nodejs --git https://github.com/openshift/nodejs-ex.git
```

### perl

Build and run Perl applications on CentOS 7. For more information about using this builder image, including OpenShift considerations, see https://github.com/sclorg/s2i-perl-container/blob/master/5.26/README.md.

```sh
  odo create perl --git https://github.com/openshift/dancer-ex.git
```

### php

Build and run PHP applications on CentOS 7. For more information about using this builder image, including OpenShift considerations, see https://github.com/sclorg/s2i-php-container/blob/master/7.1/README.md.

```sh
  odo create php --git https://github.com/openshift/cakephp-ex.git
```

### python

Build and run Python applications on CentOS 7. For more information about using this builder image, including OpenShift considerations, see https://github.com/sclorg/s2i-python-container/blob/master/3.6/README.md.

```sh
  odo create python --git https://github.com/openshift/django-ex.git
```

### ruby

Build and run Ruby applications on CentOS 7. For more information about using this builder image, including OpenShift considerations, see https://github.com/sclorg/s2i-ruby-container/blob/master/2.5/README.md.

```sh
  odo create ruby --git https://github.com/openshift/ruby-ex.git
```

### wildfly

Build and run WildFly applications on CentOS 7. For more information about using this builder image, including OpenShift considerations, see https://github.com/openshift-s2i/s2i-wildfly/blob/master/README.md.

```sh
  odo create wildfly --git https://github.com/openshift/openshift-jee-sample.git
```

## Binary example

### java 

Java can be used to deploy binary artifact, for example:

```sh
  git clone https://github.com/spring-projects/spring-petclinic.git
  cd spring-petclinic
  mvn package
  odo create java test3 --binary target/*.jar
  odo push
```

### wildfly

WildFly can deploy a binary application.

```sh
  wget -O example.war 'https://github.com/appuio/hello-world-war/blob/master/repo/ch/appuio/hello-world-war/1.0.0/hello-world-war-1.0.0.war?raw=true'
  odo create wildfly --binary example.war
```