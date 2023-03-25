/*
Copyright 2016 The Kubernetes Authors.

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

package sslpassthrough

import (
	"testing"

	api "k8s.io/api/core/v1"
	extensions "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

)

func buildIngress() *extensions.Ingress {
	return &extensions.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "foo",
			Namespace: api.NamespaceDefault,
		},
		Spec: extensions.IngressSpec{
			DefaultBackend: &extensions.IngressBackend{
				Service: &extensions.IngressServiceBackend{
					Name: "default-backend",
					Port: extensions.ServiceBackendPort{Number: 80},
				},
			},
		},
	}
}

func TestParseAnnotations(t *testing.T) {
	ing := buildIngress()

	_, err := NewParser().Parse(ing)
	if err == nil {
		t.Errorf("unexpected error: %v", err)
	}

	data := map[string]string{}
	data[passthroughAnn] = "true"
	data[httpPortAnn] = "8000"
	ing.SetAnnotations(data)
	// test ingress using the annotation without a TLS section
	_, err = NewParser().Parse(ing)
	if err != nil {
		t.Errorf("unexpected error parsing ingress with sslpassthrough")
	}

	// test with a valid host
	ing.Spec.TLS = []extensions.IngressTLS{
		{
			Hosts: []string{"foo.bar.com"},
		},
	}
	i, err := NewParser().Parse(ing)
	if err != nil {
		t.Errorf("expected error parsing ingress with sslpassthrough")
	}
	cfg, ok := i.(*Config)
	if !ok {
		t.Errorf("expected a sslpassthrough.Config type")
	}
	if !cfg.HasSSLPassthrough {
		t.Errorf("expected true but false returned")
	}
	if cfg.HTTPPort != 8000 {
		t.Errorf("expected 8000 but %v returned", cfg.HTTPPort)
	}
}
