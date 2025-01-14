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

package proxy

import (
	"testing"

	api "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
}

func (m mockBackend) GetDefaultBackend() defaults.Backend {
	return defaults.Backend{
		ProxyConnectTimeout:   10,
		ProxySendTimeout:      15,
		ProxyReadTimeout:      20,
		ProxyBufferSize:       "10k",
		ProxyBodySize:         "3k",
		ProxyNextUpstream:     "error",
		ProxyPassParams:       "nocanon keepalive=On",
		ProxyRequestBuffering: "on",
	}
}

func TestProxy(t *testing.T) {
	ing := buildIngress()

	data := map[string]string{}
	data[connect] = "1"
	data[send] = "2"
	data[read] = "3"
	data[bufferSize] = "1k"
	data[bodySize] = "2k"
	data[nextUpstream] = "off"
	data[passParams] = "smax=5 max=10"
	data[requestBuffering] = "off"
	ing.SetAnnotations(data)

	i, err := NewParser(mockBackend{}).Parse(ing)
	if err != nil {
		t.Fatalf("unexpected error parsing a valid")
	}
	p, ok := i.(*Configuration)
	if !ok {
		t.Fatalf("expected a Configuration type")
	}
	if p.ConnectTimeout != 1 {
		t.Errorf("expected 1 as connect-timeout but returned %v", p.ConnectTimeout)
	}
	if p.SendTimeout != 2 {
		t.Errorf("expected 2 as send-timeout but returned %v", p.SendTimeout)
	}
	if p.ReadTimeout != 3 {
		t.Errorf("expected 3 as read-timeout but returned %v", p.ReadTimeout)
	}
	if p.BufferSize != "1k" {
		t.Errorf("expected 1k as buffer-size but returned %v", p.BufferSize)
	}
	if p.BodySize != "2k" {
		t.Errorf("expected 2k as body-size but returned %v", p.BodySize)
	}
	if p.NextUpstream != "off" {
		t.Errorf("expected off as next-upstream but returned %v", p.NextUpstream)
	}
	if p.PassParams != "smax=5 max=10" {
		t.Errorf("expected \"smax=5 max=10\" as pass-params but returned \"%v\"", p.PassParams)
	}
	if p.RequestBuffering != "off" {
		t.Errorf("expected off as request-buffering but returned %v", p.RequestBuffering)
	}
}

func TestProxyWithNoAnnotation(t *testing.T) {
	ing := buildIngress()

	data := map[string]string{}
	ing.SetAnnotations(data)

	i, err := NewParser(mockBackend{}).Parse(ing)
	if err != nil {
		t.Fatalf("unexpected error parsing a valid")
	}
	p, ok := i.(*Configuration)
	if !ok {
		t.Fatalf("expected a Configuration type")
	}
	if p.ConnectTimeout != 10 {
		t.Errorf("expected 10 as connect-timeout but returned %v", p.ConnectTimeout)
	}
	if p.SendTimeout != 15 {
		t.Errorf("expected 15 as send-timeout but returned %v", p.SendTimeout)
	}
	if p.ReadTimeout != 20 {
		t.Errorf("expected 20 as read-timeout but returned %v", p.ReadTimeout)
	}
	if p.BufferSize != "10k" {
		t.Errorf("expected 10k as buffer-size but returned %v", p.BufferSize)
	}
	if p.BodySize != "3k" {
		t.Errorf("expected 3k as body-size but returned %v", p.BodySize)
	}
	if p.NextUpstream != "error" {
		t.Errorf("expected error as next-upstream but returned %v", p.NextUpstream)
	}
	if p.PassParams != "nocanon keepalive=On" {
		t.Errorf("expected \"nocanon keepalive=On\" as pass-params but returned \"%v\"", p.PassParams)
	}
	if p.RequestBuffering != "on" {
		t.Errorf("expected on as request-buffering but returned %v", p.RequestBuffering)
	}
}
