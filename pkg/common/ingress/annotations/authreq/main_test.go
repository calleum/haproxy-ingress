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

package authreq

import (
	"fmt"
	"reflect"
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

	tests := []struct {
		title     string
		url       string
		signinURL string
		method    string
		expErr    bool
	}{
		{"empty", "", "", "", true},
		{"no scheme", "bar", "bar", "", true},
		{"invalid host", "http://", "http://", "", true},
		{"invalid host (multiple dots)", "http://foo..bar.com", "http://foo..bar.com", "", true},
		{"valid URL", "http://bar.foo.com/external-auth", "http://bar.foo.com/external-auth", "", false},
		{"valid URL - send body", "http://foo.com/external-auth", "http://foo.com/external-auth", "POST", false},
		{"valid URL - send body", "http://foo.com/external-auth", "http://foo.com/external-auth", "GET", false},
	}

	for _, test := range tests {
		data[authURL] = test.url
		data[authSigninURL] = test.signinURL
		data[authMethod] = fmt.Sprintf("%v", test.method)

		i, err := NewParser().Parse(ing)
		if test.expErr {
			if err == nil {
				t.Errorf("%v: expected error but retuned nil", test.title)
			}
			continue
		}
		u, ok := i.(*External)
		if !ok {
			t.Errorf("%v: expected an External type", test.title)
		}
		if u.URL != test.url {
			t.Errorf("%v: expected \"%v\" but \"%v\" was returned", test.title, test.url, u.URL)
		}
		if u.SigninURL != test.signinURL {
			t.Errorf("%v: expected \"%v\" but \"%v\" was returned", test.title, test.signinURL, u.SigninURL)
		}
		if u.Method != test.method {
			t.Errorf("%v: expected \"%v\" but \"%v\" was returned", test.title, test.method, u.Method)
		}
	}
}

func TestHeaderAnnotations(t *testing.T) {
	ing := buildIngress()

	data := map[string]string{}
	ing.SetAnnotations(data)

	tests := []struct {
		title         string
		url           string
		headers       string
		parsedHeaders []string
		expErr        bool
	}{
		{"single header", "http://goog.url", "h1", []string{"h1"}, false},
		{"nothing", "http://goog.url", "", []string{}, false},
		{"spaces", "http://goog.url", "  ", []string{}, false},
		{"two headers", "http://goog.url", "1,2", []string{"1", "2"}, false},
		{"two headers and empty entries", "http://goog.url", ",1,,2,", []string{"1", "2"}, false},
		{"header with spaces", "http://goog.url", "1 2", []string{}, true},
		{"header with other bad symbols", "http://goog.url", "1+2", []string{}, true},
	}

	for _, test := range tests {
		data[authURL] = test.url
		data[authHeaders] = test.headers
		data[authMethod] = "GET"

		i, err := NewParser().Parse(ing)
		if test.expErr {
			if err == nil {
				t.Errorf("%v: expected error but retuned nil", err.Error())
			}
			continue
		}

		t.Log(i)
		u, ok := i.(*External)
		if !ok {
			t.Errorf("%v: expected an External type", test.title)
			continue
		}

		if !reflect.DeepEqual(u.ResponseHeaders, test.parsedHeaders) {
			t.Errorf("%v: expected \"%v\" but \"%v\" was returned", test.title, test.headers, u.ResponseHeaders)
		}
	}
}
