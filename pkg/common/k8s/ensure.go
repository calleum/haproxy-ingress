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

package k8s

import (
    "context"
	api "k8s.io/api/core/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func EnsureSecret(cl kubernetes.Interface, secret *api.Secret) (*api.Secret, error) {
    ctx := context.Background()
	s, err := cl.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			return cl.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
		}
		return nil, err
	}
	return s, nil
}

func EnsureIngress(cl kubernetes.Interface, ingress *networking.Ingress) (*networking.Ingress, error) {
    ctx := context.Background()
	s, err := cl.NetworkingV1().Ingresses(ingress.Namespace).Update(ctx, ingress, metav1.UpdateOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return cl.NetworkingV1().Ingresses(ingress.Namespace).Create(ctx, ingress, metav1.CreateOptions{})
		}
		return nil, err
	}
	return s, nil
}

func EnsureService(cl kubernetes.Interface, service *core.Service) (*core.Service, error) {
    ctx := context.Background()
	s, err := cl.CoreV1().Services(service.Namespace).Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return cl.CoreV1().Services(service.Namespace).Create(ctx, service, metav1.CreateOptions{})
		}
		return nil, err
	}
	return s, nil
}
