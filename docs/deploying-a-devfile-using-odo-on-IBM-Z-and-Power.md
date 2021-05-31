---
# Page settings
layout: default
keywords:
comments: false

# Hero section
title: Deploy a devfile on IBM Z & Power 
description: provide the doc to introduce how to rue devfiles on IBM Z & Power platform.

# Micro navigation
micro_nav: true

# Page navigation
page_nav:
    prev:
        content: Deploying your first application using odo
        url: '/docs/deploying-a-devfile-using-odo'
    next:
        content: Devfile file reference
        url: '/file-reference/'
---
# Using odo on IBM-Z and Power

## Introduction to devfile

What is a devfile?

A [devfile](https://redhat-developer.github.io/devfile/) is a portable file that describes your development environment. It allows reproducing a *portable* development environment without the need of reconfiguration.

With a devfile you can describe:

  - Development components such as container definition for build and application runtimes

  - A list of pre-defined commands that can be run

  - Projects to initially clone

odo takes this devfile and uses it to create a workspace of multiple containers running on Kubernetes or OpenShift.

Devfiles are YAML files with a defined [schema](https://devfile.github.io/devfile/_attachments/api-reference.html).

## odo and devfile

odo can now create components from devfiles as recorded in registries. odo automatically consults the [default registry](https://github.com/odo-devfiles/registry) but users can also add their own registries. Devfiles contribute new component types that users can pull to begin development immediately.

An example deployment scenario:

1.  `odo create` will consult the recorded devfile registries to offer the user a selection of available component types and pull down the associated `devfile.yaml` file

2.  `odo push` parses and then deploys the component in the following order:
    
    1.  Parses and validates the YAML file
    
    2.  Deploys the development environment to your Kubernetes or OpenShift cluster
    
    3.  Synchronizes your source code to the containers
    
    4.  Executes any prerequisite commands

## Deploying your first devfile on IBM Z & Power

Since the DefaultDevfileRegistry doesn’t support IBM Z & Power now, you will need to create a secure private DevfileRegistry first. To create a new secure private DevfileRegistry , please check the doc [secure registry](https://github.com/openshift/odo/blob/main/docs/public/secure-registry.adoc).

The images can be used for devfiles on IBM Z & Power

| Language    | Devfile Name     | Description                        | Image Source                                                       | Supported Platform |
| ----------- | ---------------- | ---------------------------------- | ------------------------------------------------------------------ | ------------------ |
| Java        | java-maven       | Upstream Maven and OpenJDK 11      | registry.redhat.io/codeready-workspaces/plugin-java11-openj9-rhel8 | s390x, ppc64le     |
| Java        | java-openliberty | Open Liberty microservice in Java  | registry.redhat.io/codeready-workspaces/plugin-java11-openj9-rhel8 | s390x, ppc64le     |
| Java        | java-quarkus     | Upstream Quarkus with Java+GraalVM | registry.redhat.io/codeready-workspaces/plugin-java8-openj9-rhel8  | s390x, ppc64le     |
| Java        | java-springboot  | Spring Boot® using Java            | registry.redhat.io/codeready-workspaces/plugin-java11-openj9-rhel8 | s390x, ppc64le     |
| Vert.x Java | java-vertx       | Upstream Vert.x using Java         | registry.redhat.io/codeready-workspaces/plugin-java11-openj9-rhel8 | s390x, ppc64le     |
| Node.JS     | nodejs           | Stack with NodeJS 12               | registry.redhat.io/codeready-workspaces/plugin-java8-openj9-rhel8  | s390x, ppc64le     |
| Python      | python           | Python Stack with Python 3.7       | registry.redhat.io/codeready-workspaces/plugin-java8-openj9-rhel8  | s390x, ppc64le     |
| Django      | python-django    | Python3.7 with Django              | registry.redhat.io/codeready-workspaces/plugin-java8-openj9-rhel8  | s390x, ppc64le     |

> **Note**
> 
> Access to the Red Hat registry is required to use these images on IBM Power Systems & IBM Z.

Steps to use devfiles can be found in the doc [deploy your first devfile](https://github.com/openshift/odo/blob/main/docs/public/deploying-a-devfile-using-odo.adoc#deploying-your-first-devfile).
