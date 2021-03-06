/*
Copyright The Kubernetes Authors.

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

package v1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1 "xworld.cn/pkg/apis/xworld/v1"
)

// XServerLister helps list XServers.
type XServerLister interface {
	// List lists all XServers in the indexer.
	List(selector labels.Selector) (ret []*v1.XServer, err error)
	// XServers returns an object that can list and get XServers.
	XServers(namespace string) XServerNamespaceLister
	XServerListerExpansion
}

// xServerLister implements the XServerLister interface.
type xServerLister struct {
	indexer cache.Indexer
}

// NewXServerLister returns a new XServerLister.
func NewXServerLister(indexer cache.Indexer) XServerLister {
	return &xServerLister{indexer: indexer}
}

// List lists all XServers in the indexer.
func (s *xServerLister) List(selector labels.Selector) (ret []*v1.XServer, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.XServer))
	})
	return ret, err
}

// XServers returns an object that can list and get XServers.
func (s *xServerLister) XServers(namespace string) XServerNamespaceLister {
	return xServerNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// XServerNamespaceLister helps list and get XServers.
type XServerNamespaceLister interface {
	// List lists all XServers in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1.XServer, err error)
	// Get retrieves the XServer from the indexer for a given namespace and name.
	Get(name string) (*v1.XServer, error)
	XServerNamespaceListerExpansion
}

// xServerNamespaceLister implements the XServerNamespaceLister
// interface.
type xServerNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all XServers in the indexer for a given namespace.
func (s xServerNamespaceLister) List(selector labels.Selector) (ret []*v1.XServer, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.XServer))
	})
	return ret, err
}

// Get retrieves the XServer from the indexer for a given namespace and name.
func (s xServerNamespaceLister) Get(name string) (*v1.XServer, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("xserver"), name)
	}
	return obj.(*v1.XServer), nil
}
