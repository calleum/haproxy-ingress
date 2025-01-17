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

package auth

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"

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

type (
	mockConfig struct {
	}
	mockSecret struct {
	}
)

func (m mockConfig) GetFullResourceName(name, currentNamespace string) string {
	return fmt.Sprintf("%v/%v", currentNamespace, name)
}

func (m mockSecret) GetSecret(name string) (*api.Secret, error) {
	if name != "default/demo-secret" {
		return nil, errors.Errorf("there is no secret with name %v", name)
	}

	return &api.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: api.NamespaceDefault,
			Name:      "demo-secret",
		},
		Data: map[string][]byte{"auth": []byte("foo:$apr1$OFG3Xybp$ckL0FHDAkoXYIlH9.cysT0")},
	}, nil
}

func TestIngressWithoutAuth(t *testing.T) {
	ing := buildIngress()
	_, dir, _ := dummySecretContent(t)
	defer os.RemoveAll(dir)
	_, err := NewParser(dir, mockConfig{}, mockSecret{}).Parse(ing)
	if err == nil {
		t.Error("Expected error with ingress without annotations")
	}
}

func TestIngressAuth(t *testing.T) {
	ing := buildIngress()

	data := map[string]string{}
	data[authType] = "basic"
	data[authSecret] = "demo-secret"
	data[authRealm] = "-realm-"
	ing.SetAnnotations(data)

	_, dir, _ := dummySecretContent(t)
	defer os.RemoveAll(dir)

	i, err := NewParser(dir, mockConfig{}, mockSecret{}).Parse(ing)
	if err != nil {
		t.Errorf("Uxpected error with ingress: %v", err)
	}
	auth, ok := i.(*BasicDigest)
	if !ok {
		t.Errorf("expected a BasicDigest type")
	}
	if auth.Type != "basic" {
		t.Errorf("Expected basic as auth type but returned %s", auth.Type)
	}
	if auth.Realm != "-realm-" {
		t.Errorf("Expected -realm- as realm but returned %s", auth.Realm)
	}
	if !auth.Secured {
		t.Errorf("Expected true as secured but returned %v", auth.Secured)
	}
}

func TestIngressAuthWithoutSecret(t *testing.T) {
	ing := buildIngress()

	data := map[string]string{}
	data[authType] = "basic"
	data[authSecret] = "invalid-secret"
	data[authRealm] = "-realm-"
	ing.SetAnnotations(data)

	_, dir, _ := dummySecretContent(t)
	defer os.RemoveAll(dir)

	s, _ := NewParser(dir, mockConfig{}, mockSecret{}).Parse(ing)
	secret := s.(*BasicDigest)
	if secret.Type != "basic" {
		t.Errorf("expected auth of type basic")
	}
	if secret.File != "" {
		t.Errorf("expected clean filename with invalid secret name")
	}
}

func dummySecretContent(t *testing.T) (string, string, *api.Secret) {
	dir, err := ioutil.TempDir("", fmt.Sprintf("%v", time.Now().Unix()))
	if err != nil {
		t.Error(err)
	}

	tmpfile, err := ioutil.TempFile("", "example-")
	if err != nil {
		t.Error(err)
	}
	defer tmpfile.Close()
	s, _ := mockSecret{}.GetSecret("default/demo-secret")
	return tmpfile.Name(), dir, s
}

func TestDumpSecret(t *testing.T) {
	tmpfile, dir, s := dummySecretContent(t)
	defer os.RemoveAll(dir)

	sd := s.Data
	s.Data = nil

	err := dumpSecret(tmpfile, s)
	if err == nil {
		t.Errorf("Expected error with secret without auth")
	}

	s.Data = sd
	err = dumpSecret(tmpfile, s)
	if err != nil {
		t.Errorf("Unexpected error creating htpasswd file %v: %v", tmpfile, err)
	}
}
