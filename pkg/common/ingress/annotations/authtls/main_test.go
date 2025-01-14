/*
Copyright 2015 The Kubernetes Authors.

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

package authtls

import (
	"testing"

	api "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestAnnotations(t *testing.T) {
	ing := buildIngress()

	data := map[string]string{}
	ing.SetAnnotations(data)
	/*
		tests := []struct {
			title    string
			url      string
			method   string
			sendBody bool
			expErr   bool
		}{
			{"empty", "", "", false, true},
			{"no scheme", "bar", "", false, true},
			{"invalid host", "http://", "", false, true},
			{"invalid host (multiple dots)", "http://foo..bar.com", "", false, true},
			{"valid URL", "http://bar.foo.com/external-auth", "", false, false},
			{"valid URL - send body", "http://foo.com/external-auth", "POST", true, false},
			{"valid URL - send body", "http://foo.com/external-auth", "GET", true, false},
		}

		for _, test := range tests {
			data[authTLSSecret] = ""
			test.title

				u, err := ParseAnnotations(ing)

				if test.expErr {
					if err == nil {
						t.Errorf("%v: expected error but retuned nil", test.title)
					}
					continue
				}

				if u.URL != test.url {
					t.Errorf("%v: expected \"%v\" but \"%v\" was returned", test.title, test.url, u.URL)
				}
				if u.Method != test.method {
					t.Errorf("%v: expected \"%v\" but \"%v\" was returned", test.title, test.method, u.Method)
				}
				if u.SendBody != test.sendBody {
					t.Errorf("%v: expected \"%v\" but \"%v\" was returned", test.title, test.sendBody, u.SendBody)
				}
		}*/
}
