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

package rewrite

import (
	"testing"

	api "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jcmoraisjr/haproxy-ingress/pkg/common/ingress/defaults"
)

const (
	defRoute = "/demo"
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
	redirect bool
}

func (m mockBackend) GetDefaultBackend() defaults.Backend {
	return defaults.Backend{SSLRedirect: m.redirect}
}

func TestWithoutAnnotations(t *testing.T) {
	ing := buildIngress()
	_, err := NewParser(mockBackend{}).Parse(ing)
	if err != nil {
		t.Errorf("unexpected error with ingress without annotations: %v", err)
	}
}

func TestRedirect(t *testing.T) {
	ing := buildIngress()

	data := map[string]string{}
	data[rewriteTo] = defRoute
	ing.SetAnnotations(data)

	i, err := NewParser(mockBackend{}).Parse(ing)
	if err != nil {
		t.Errorf("Unexpected error with ingress: %v", err)
	}
	redirect, ok := i.(*Redirect)
	if !ok {
		t.Errorf("expected a Redirect type")
	}
	if redirect.Target != defRoute {
		t.Errorf("Expected %v as redirect but returned %s", defRoute, redirect.Target)
	}
}

func TestSSLRedirect(t *testing.T) {
	ing := buildIngress()

	data := map[string]string{}
	data[rewriteTo] = defRoute
	ing.SetAnnotations(data)

	i, _ := NewParser(mockBackend{true}).Parse(ing)
	redirect, ok := i.(*Redirect)
	if !ok {
		t.Errorf("expected a Redirect type")
	}
	if !redirect.SSLRedirect {
		t.Errorf("Expected true but returned false")
	}

	data[sslRedirect] = "false"
	ing.SetAnnotations(data)

	i, _ = NewParser(mockBackend{false}).Parse(ing)
	redirect, ok = i.(*Redirect)
	if !ok {
		t.Errorf("expected a Redirect type")
	}
	if redirect.SSLRedirect {
		t.Errorf("Expected false but returned true")
	}
}

func TestForceSSLRedirect(t *testing.T) {
	ing := buildIngress()

	data := map[string]string{}
	data[rewriteTo] = defRoute
	ing.SetAnnotations(data)

	i, _ := NewParser(mockBackend{true}).Parse(ing)
	redirect, ok := i.(*Redirect)
	if !ok {
		t.Errorf("expected a Redirect type")
	}
	if redirect.ForceSSLRedirect {
		t.Errorf("Expected false but returned true")
	}

	data[forceSSLRedirect] = "true"
	ing.SetAnnotations(data)

	i, _ = NewParser(mockBackend{false}).Parse(ing)
	redirect, ok = i.(*Redirect)
	if !ok {
		t.Errorf("expected a Redirect type")
	}
	if !redirect.ForceSSLRedirect {
		t.Errorf("Expected true but returned false")
	}
}
func TestAppRoot(t *testing.T) {
	ing := buildIngress()

	data := map[string]string{}
	data[appRoot] = "/app1"
	ing.SetAnnotations(data)

	i, _ := NewParser(mockBackend{true}).Parse(ing)
	redirect, ok := i.(*Redirect)
	if !ok {
		t.Errorf("expected a App Context")
	}
	if redirect.AppRoot != "/app1" {
		t.Errorf("Unexpected value got in AppRoot")
	}

}
