---
# Page settings
layout: default
keywords:
comments: false

# Hero section
title: Using sample applications
description: A list of sample applications to be used with odo

# Micro navigation
micro_nav: true
---
`odo` offers partial compatibility with any language or runtime listed within the OpenShift catalog of component types. For example:

``` terminal
NAME        PROJECT       TAGS
dotnet      openshift     3.1,latest
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

> **Note**
> 
> For `odo` Java and Node.js are the officially supported component types. Run `odo catalog list components` to verify the officially supported component types.

To access the component over the web, create a URL using `odo url create`.

# Examples from Git repositories

## httpd

This example helps build and serve static content using httpd on CentOS 7. For more information about using this builder image, including OpenShift considerations, see the [Apache HTTP Server container image repository](https://github.com/sclorg/httpd-container/blob/master/2.4/root/usr/share/container-scripts/httpd/README.md).

``` terminal
$ odo create httpd --git https://github.com/openshift/httpd-ex.git
```

## java

This example helps build and run fat JAR Java applications on CentOS 7. For more information about using this builder image, including OpenShift considerations, see the [Java S2I Builder image](https://github.com/fabric8io-images/s2i/blob/master/README.md).

``` terminal
$ odo create java --git https://github.com/spring-projects/spring-petclinic.git
```

## nodejs

Build and run Node.js applications on CentOS 7. For more information about using this builder image, including OpenShift considerations, see the [Node.js 8 container image](https://github.com/sclorg/s2i-nodejs-container/blob/master/8/README.md).

``` terminal
$ odo create nodejs --git https://github.com/openshift/nodejs-ex.git
```

## perl

This example helps build and run Perl applications on CentOS 7. For more information about using this builder image, including OpenShift considerations, see the [Perl 5.26 container image](https://github.com/sclorg/s2i-perl-container/blob/master/5.26/README.md).

``` terminal
$ odo create perl --git https://github.com/openshift/dancer-ex.git
```

## php

This example helps build and run PHP applications on CentOS 7. For more information about using this builder image, including OpenShift considerations, see the [PHP 7.1 Docker image](https://github.com/sclorg/s2i-php-container/blob/master/7.1/README.md).

``` terminal
$ odo create php --git https://github.com/openshift/cakephp-ex.git
```

## python

This example helps build and run Python applications on CentOS 7. For more information about using this builder image, including OpenShift considerations, see the [Python 3.6 container image](https://github.com/sclorg/s2i-python-container/blob/master/3.6/README.md).

``` terminal
$ odo create python --git https://github.com/openshift/django-ex.git
```

## ruby

This example helps build and run Ruby applications on CentOS 7. For more information about using this builder image, including OpenShift considerations, see [Ruby 2.5 container image](https://github.com/sclorg/s2i-ruby-container/blob/master/2.5/README.md).

``` terminal
$ odo create ruby --git https://github.com/openshift/ruby-ex.git
```

# Binary examples

## java

Java can be used to deploy a binary artifact as follows:

``` terminal
$ git clone https://github.com/spring-projects/spring-petclinic.git
$ cd spring-petclinic
$ mvn package
$ odo create java test3 --binary target/*.jar
$ odo push
```
