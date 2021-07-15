---
title: Basics
sidebar_position: 3
---
# odo concepts

odo abstracts Kubernetes concepts into developer focussed terminology. In this document, we will take a look at each of these concepts and also their Kubernetes equivalent.

### Concepts

* Component
  A component is a microservice. Each component can be run and deployed separately. Examples of components: Warehouse API Backend, Inventory API, Web Frontend, Payment Backend
  > A component in odo is an equivalent of a deployment or deploymentconfig in Kubernetes. Creating a new component in odo will create a deployment in Kubernetes or for git and binary based component creation, it will create deploymentconfig.
* Application
  An application consists of multiple components which may span over multiple projects, and work individually to build the entire application. Examples of applications: e-Shop, Hotel Reservation System, Online Booking
  > An application can be considered as an equivalent of labels in Kubernetes that help in grouping a set of resources.
* Project
  A project is your source code, tests, and libraries organized in a separate single unit.
  > A project in odo is an equivalent of a namespace in Kubernetes. Creating a project in odo will create a namespace in Kubernetes with the same name.
* Context
  A context is the directory where the source code, tests, and libraries for your component resides. A single context can only contain a single component.
* URL
  A URL exposes your component to be accessed outside the cluster.
  > A URL is an equivalent of a service in Kubernetes.
* Storage
  A storage volume is [PVC](https://kubernetes.io/docs/concepts/storage/volumes/#persistentvolumeclaim) which is a way for you to "claim" persistent storage without knowing the details of the environment. Storage volume can persist data across restarts and rebuilds of a component.
  > A storage volume is an equivalent of PVC in Kubernetes. Creating a storage with odo will create a PVC resource in Kubernetes with the same name.
* Service
  A service is another microservice, or a Kubernetes Custom Resource that your component connects to or depends on. Example of services: MariaDB, MySQL
  > A service created with odo creates multiple Kubernetes resources to establish a proper linking between the source component and target microservice, or 
* Devfile
  A portable YAML file responsible for your entire reproducible development environment. See [Devfile](../architecture/devfile.md) to know more about devfile.
  > A devfile can be considered similar to a manifest file in Kubernetes, but there is no equivalent of it in Kubernetes.
  

### Component vs. Application
A component is a microservice, while an application is a group of microservices. To make a microservice belong to an application, all the resources belonging to the microservice are assigned a label.

### odo services vs. Kubernetes services
odo service helps in connecting one microservice to another, while a Kubernetes service helps in making a microservice accessible outside the cluster.
Creating an odo service will create a set of resources to establish a successful connection between the two microservices.