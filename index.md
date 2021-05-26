---
# Page settings
layout: homepage
keywords:

# Hero section
title: odo 
description: "odo is a fast, iterative, and straightforward CLI tool for developers who write, build, and deploy applications on Kubernetes and OpenShift.<br><br>Existing tools such as kubectl and oc are more operations-focused and require a deep-understanding of Kubernetes and OpenShift concepts. odo abstracts away complex Kubernetes and OpenShift concepts for the developer."
project: 
asciinema: '<script id="asciicast-uIcSZvdbrFKKeH2sqrLsFdXym" src="https://asciinema.org/a/uIcSZvdbrFKKeH2sqrLsFdXym.js" async></script>'

buttons:
    - content: Installing odo
      url: '/docs/installing-odo'
      external_url: false
    - icon: github
      content: GitHub
      url: 'https://github.com/openshift/odo'
      external_url: true

# Author box
author:
    title: Official odo Devfile Registry
    title_url: 'https://github.com/odo-devfiles/registry'
    external_url: true
    description: "odo has examples for multiple languages and frameworks. However, with Devfile, you can take any language or framework and deploy it."
    languages:
      - title: Java (Maven)
        url: https://github.com/odo-devfiles/registry/blob/master/devfiles/java-maven/devfile.yaml
      - title: Java (Springboot)
        url: https://github.com/odo-devfiles/registry/blob/master/devfiles/java-springboot/devfile.yaml
      - title: Node.JS
        url: https://github.com/odo-devfiles/registry/blob/master/devfiles/nodejs/devfile.yaml

# Micro navigation
micro_nav: true

# Grid navigation
grid_navigation:
    - title: Installing odo
      excerpt: Installing odo on macOS, Linux and Windows
      cta: Read more
      url: 'docs/installing-odo'

    - title: Understanding odo
      excerpt: Understanding the concepts of odo
      cta: Read more
      url: 'docs/understanding-odo'

    - title: Deploying your first application using odo
      excerpt: Learn how to deploy an application using odo and Devfile
      cta: Read more
      url: 'docs/deploying-a-devfile-using-odo'

    - title: Devfile file reference
      excerpt: An overview of the Devfile yaml file format, learn how to customize your devfile.yaml file
      cta: Read more
      url: 'file-reference/'

    - title: Debugging applications in odo
      excerpt: Learn how to debug an application in odo CLI and IDE
      cta: Read more
      url: 'docs/debugging-using-devfile'

    - title: Setup the minikube environment
      excerpt: Setup a Kubernetes cluster that odo can be used with
      cta: Read more
      url: 'docs/installing-and-configuring-minikube-environment'

    - title: Introduction to Operators
      excerpt: Deploying an Operator from Operator Hub using odo.
      cta: Read more
      url: 'docs/operator-hub'

    - title: Java OpenLiberty with PostgreSQL
      excerpt: Binding a Java microservices JPA application to an in-cluster Operator-managed PostgreSQL database on minikube
      cta: Read more
      url: 'docs/deploying-java-app-with-database'

    - title: Installing Operators on minikube
      excerpt: Installing etcd Operator on minikube
      cta: Read more
      url: 'docs/operators-on-minikube'

    - title: Installing Service Binding Operator
      excerpt: Installing Service Binding Operator on OpenShift and Kubernetes
      cta: Read more
      url: 'docs/install-service-binding-operator'

    - title: Migrating S2I (Source-to-Image) components to Devfile components
      excerpt: Use odo's built-in tool to convert your S2I deployment to devfile
      cta: Read more
      url: 'docs/s2i-to-devfile'
    
    - title: Breaking changes in odo 2.2.0
      excerpt: Explaining the breaking changing that have been done in odo.2.2.0
      cta: Read more
      url: 'docs/breaking-changes-in-odo-2.2'

    - title: Setting up a secure Devfile registry
      excerpt: Learn how to setup a secure private registry that only you or your team can access
      cta: Read more
      url: 'docs/secure-registry'

    - title: Using persistent storage
      excerpt: Setup a storage volume for persistent data
      cta: Read more
      url: 'docs/using-storage'

    - title: Using Devfile lifecycle events within odo
      excerpt: Use Devfile lifecycle events to control each aspect of your component deployment
      cta: Read more
      url: 'docs/using-devfile-lifecycle-events'

    - title: Using the odo.dev.push.path related attributes
      excerpt: Push only the specified files and folders to the component.
      cta: Read more
      url: 'docs/using-devfile-odo.dev.push.path-attribute'

    - title: Architecture of odo
      excerpt: A general overview of the odo architecture
      cta: Read more
      url: 'docs/odo-architecture'

    - title: Managing environment variables
      excerpt: Manipulate both config and preference files to your liking
      cta: Read more
      url: 'docs/managing-environment-variables-in-odo'

    - title: Configuring the odo CLI
      excerpt: Configure your terminal for autocompletion
      cta: Read more
      url: 'docs/configuring-the-odo-cli'

    - title: odo CLI reference
      excerpt: An overview of all the CLI commands related to odo
      cta: Read more
      url: 'docs/odo-cli-reference'
---
