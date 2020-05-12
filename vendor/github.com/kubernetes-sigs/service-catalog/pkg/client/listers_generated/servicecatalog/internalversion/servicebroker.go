/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package internalversion

import (
	servicecatalog "github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ServiceBrokerLister helps list ServiceBrokers.
type ServiceBrokerLister interface {
	// List lists all ServiceBrokers in the indexer.
	List(selector labels.Selector) (ret []*servicecatalog.ServiceBroker, err error)
	// ServiceBrokers returns an object that can list and get ServiceBrokers.
	ServiceBrokers(namespace string) ServiceBrokerNamespaceLister
	ServiceBrokerListerExpansion
}

// serviceBrokerLister implements the ServiceBrokerLister interface.
type serviceBrokerLister struct {
	indexer cache.Indexer
}

// NewServiceBrokerLister returns a new ServiceBrokerLister.
func NewServiceBrokerLister(indexer cache.Indexer) ServiceBrokerLister {
	return &serviceBrokerLister{indexer: indexer}
}

// List lists all ServiceBrokers in the indexer.
func (s *serviceBrokerLister) List(selector labels.Selector) (ret []*servicecatalog.ServiceBroker, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*servicecatalog.ServiceBroker))
	})
	return ret, err
}

// ServiceBrokers returns an object that can list and get ServiceBrokers.
func (s *serviceBrokerLister) ServiceBrokers(namespace string) ServiceBrokerNamespaceLister {
	return serviceBrokerNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ServiceBrokerNamespaceLister helps list and get ServiceBrokers.
type ServiceBrokerNamespaceLister interface {
	// List lists all ServiceBrokers in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*servicecatalog.ServiceBroker, err error)
	// Get retrieves the ServiceBroker from the indexer for a given namespace and name.
	Get(name string) (*servicecatalog.ServiceBroker, error)
	ServiceBrokerNamespaceListerExpansion
}

// serviceBrokerNamespaceLister implements the ServiceBrokerNamespaceLister
// interface.
type serviceBrokerNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ServiceBrokers in the indexer for a given namespace.
func (s serviceBrokerNamespaceLister) List(selector labels.Selector) (ret []*servicecatalog.ServiceBroker, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*servicecatalog.ServiceBroker))
	})
	return ret, err
}

// Get retrieves the ServiceBroker from the indexer for a given namespace and name.
func (s serviceBrokerNamespaceLister) Get(name string) (*servicecatalog.ServiceBroker, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(servicecatalog.Resource("servicebroker"), name)
	}
	return obj.(*servicecatalog.ServiceBroker), nil
}
