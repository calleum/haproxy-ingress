/*
Copyright 2017 The Kubernetes Authors.

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

package serviceupstream

import (
	"github.com/jcmoraisjr/haproxy-ingress/pkg/common/ingress/annotations/parser"
	networking "k8s.io/api/networking/v1"
)

const (
	annotationServiceUpstream = "ingress.kubernetes.io/service-upstream"
)

type serviceUpstream struct {
}

// NewParser creates a new serviceUpstream annotation parser
func NewParser() parser.IngressAnnotation {
	return serviceUpstream{}
}

func (s serviceUpstream) Parse(ing *networking.Ingress) (interface{}, error) {
	return parser.GetBoolAnnotation(annotationServiceUpstream, ing)
}
