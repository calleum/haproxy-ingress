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

package healthcheck

import (
	"testing"

	api "k8s.io/api/core/v1"
	extensions "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func buildIngress() *extensions.Ingress {
    svc := extensions.IngressServiceBackend{
    	Name: "default-backend",
    	Port: extensions.ServiceBackendPort{Number: 80},
    }

	defaultBackend := extensions.IngressBackend{
        Service: &svc,
	}

	return &extensions.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "foo",
			Namespace: api.NamespaceDefault,
		},
		Spec: extensions.IngressSpec{
			DefaultBackend: &extensions.IngressBackend{
                Service: &svc,
			},
			Rules: []extensions.IngressRule{
				{
					Host: "foo.bar.com",
					IngressRuleValue: extensions.IngressRuleValue{
						HTTP: &extensions.HTTPIngressRuleValue{
							Paths: []extensions.HTTPIngressPath{
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

func TestIngressHealthCheck(t *testing.T) {
	ing := buildIngress()

	data := map[string]string{}
	data[healthCheckURI] = "/foo"
	data[healthCheckAddr] = "1.2.3.4"
	data[healthCheckPort] = "8080"
	data[healthCheckInterval] = "7"
	data[healthCheckRiseCount] = "8"
	data[healthCheckFallCount] = "9"
	ing.SetAnnotations(data)

	hc, _ := NewParser().Parse(ing)
	healthCheck, ok := hc.(*Config)
	if !ok {
		t.Errorf("expected a Config type")
	}

	if healthCheck.URI != "/foo" {
		t.Errorf("expected /foo as URI but returned %v", healthCheck.URI)
	}
	if healthCheck.Addr != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4 as Addr but returned %v", healthCheck.Addr)
	}
	if healthCheck.Port != "8080" {
		t.Errorf("expected 8080 as port but returned %v", healthCheck.Port)
	}
	if healthCheck.Interval != "7" {
		t.Errorf("expected 7 as Interval but returned %v", healthCheck.Interval)
	}
	if healthCheck.RiseCount != "8" {
		t.Errorf("expected 8 as RiseCount but returned %v", healthCheck.RiseCount)
	}
	if healthCheck.FallCount != "9" {
		t.Errorf("expected 9 as FallCount but returned %v", healthCheck.FallCount)
	}
}
