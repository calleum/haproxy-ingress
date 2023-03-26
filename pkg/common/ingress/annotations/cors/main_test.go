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

package cors

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

func TestIngressCorsConfig(t *testing.T) {
	ing := buildIngress()

	data := map[string]string{}
	data[annotationCorsEnabled] = "true"
	data[annotationCorsAllowHeaders] = "DNT,X-CustomHeader, Keep-Alive,User-Agent"
	data[annotationCorsAllowCredentials] = "false"
	data[annotationCorsAllowMethods] = "PUT, GET,OPTIONS, PATCH, $nginx_version"
	data[annotationCorsAllowOrigin] = "https://origin123.test.com:4443"
	ing.SetAnnotations(data)

	corst, _ := NewParser().Parse(ing)
	nginxCors, ok := corst.(*CorsConfig)
	if !ok {
		t.Errorf("expected a Config type")
	}

	if nginxCors.CorsEnabled != true {
		t.Errorf("expected cors enabled but returned %v", nginxCors.CorsEnabled)
	}

	if nginxCors.CorsAllowHeaders != "DNT,X-CustomHeader, Keep-Alive,User-Agent" {
		t.Errorf("expected headers not found. Found %v", nginxCors.CorsAllowHeaders)
	}

	if nginxCors.CorsAllowMethods != "GET, PUT, POST, DELETE, PATCH, OPTIONS" {
		t.Errorf("expected default methods, but got  %v", nginxCors.CorsAllowMethods)
	}

	if nginxCors.CorsAllowOrigin != "https://origin123.test.com:4443" {
		t.Errorf("expected origin https://origin123.test.com:4443, but got  %v", nginxCors.CorsAllowOrigin)
	}

}
