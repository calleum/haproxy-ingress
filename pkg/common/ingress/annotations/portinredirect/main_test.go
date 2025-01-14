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

package portinredirect

import (
	"testing"

	api "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"fmt"

	"github.com/jcmoraisjr/haproxy-ingress/pkg/common/ingress/defaults"
)

func buildIngress() *networking.Ingress {
    svc := networking.IngressServiceBackend{
    	Name: "default-backend",
    	Port: networking.ServiceBackendPort{Number: 80},
    }
	defaultBackend := networking.IngressBackend{
        Service: &svc,
	}

	return &networking.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "foo",
			Namespace: api.NamespaceDefault,
		},
		Spec: networking.IngressSpec{
			DefaultBackend: &networking.IngressBackend{
                Service: &svc,
			},
			Rules: []networking.IngressRule{
				{
					Host: "foo.bar.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path:    "/foo",
									Backend: defaultBackend,
								},
							},
						},
					},
				},
			},
		},
	}
}

type mockBackend struct {
	usePortInRedirects bool
}

func (m mockBackend) GetDefaultBackend() defaults.Backend {
	return defaults.Backend{UsePortInRedirects: m.usePortInRedirects}
}

func TestPortInRedirect(t *testing.T) {
	tests := []struct {
		title   string
		usePort *bool
		def     bool
		exp     bool
	}{
		{"false - default false", newFalse(), false, false},
		{"false - default true", newFalse(), true, false},
		{"no annotation - default false", nil, false, false},
		{"no annotation - default true", nil, true, true},
		{"true - default true", newTrue(), true, true},
	}

	for _, test := range tests {
		ing := buildIngress()

		data := map[string]string{}
		if test.usePort != nil {
			data[annotation] = fmt.Sprintf("%v", *test.usePort)
		}
		ing.SetAnnotations(data)

		i, err := NewParser(mockBackend{test.def}).Parse(ing)
		if err != nil {
			t.Errorf("unexpected error parsing a valid")
		}
		p, ok := i.(bool)
		if !ok {
			t.Errorf("expected a bool type")
		}

		if p != test.exp {
			t.Errorf("%v: expected \"%v\" but \"%v\" was returned", test.title, test.exp, p)
		}
	}
}

func newTrue() *bool {
	b := true
	return &b
}

func newFalse() *bool {
	b := false
	return &b
}
