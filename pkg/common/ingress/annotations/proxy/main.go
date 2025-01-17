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
	"github.com/golang/glog"
	networking "k8s.io/api/networking/v1"
	"regexp"

	"github.com/jcmoraisjr/haproxy-ingress/pkg/common/ingress/annotations/parser"
	"github.com/jcmoraisjr/haproxy-ingress/pkg/common/ingress/resolver"
)

const (
	bodySize         = "ingress.kubernetes.io/proxy-body-size"
	connect          = "ingress.kubernetes.io/proxy-connect-timeout"
	send             = "ingress.kubernetes.io/proxy-send-timeout"
	read             = "ingress.kubernetes.io/proxy-read-timeout"
	bufferSize       = "ingress.kubernetes.io/proxy-buffer-size"
	cookiePath       = "ingress.kubernetes.io/proxy-cookie-path"
	cookieDomain     = "ingress.kubernetes.io/proxy-cookie-domain"
	nextUpstream     = "ingress.kubernetes.io/proxy-next-upstream"
	passParams       = "ingress.kubernetes.io/proxy-pass-params"
	requestBuffering = "ingress.kubernetes.io/proxy-request-buffering"
)

var (
	bodySizeRegex = regexp.MustCompile(`^(|([0-9]+[kmg]?))$`)
)

// Configuration returns the proxy timeout to use in the upstream server/s
type Configuration struct {
	BodySize         string `json:"bodySize"`
	ConnectTimeout   int    `json:"connectTimeout"`
	SendTimeout      int    `json:"sendTimeout"`
	ReadTimeout      int    `json:"readTimeout"`
	BufferSize       string `json:"bufferSize"`
	CookieDomain     string `json:"cookieDomain"`
	CookiePath       string `json:"cookiePath"`
	NextUpstream     string `json:"nextUpstream"`
	PassParams       string `json:"passParams"`
	RequestBuffering string `json:"requestBuffering"`
}

// Equal tests for equality between two Configuration types
func (l1 *Configuration) Equal(l2 *Configuration) bool {
	if l1 == l2 {
		return true
	}
	if l1 == nil || l2 == nil {
		return false
	}
	if l1.BodySize != l2.BodySize {
		return false
	}
	if l1.ConnectTimeout != l2.ConnectTimeout {
		return false
	}
	if l1.SendTimeout != l2.SendTimeout {
		return false
	}
	if l1.ReadTimeout != l2.ReadTimeout {
		return false
	}
	if l1.BufferSize != l2.BufferSize {
		return false
	}
	if l1.CookieDomain != l2.CookieDomain {
		return false
	}
	if l1.CookiePath != l2.CookiePath {
		return false
	}
	if l1.NextUpstream != l2.NextUpstream {
		return false
	}
	if l1.PassParams != l2.PassParams {
		return false
	}

	if l1.RequestBuffering != l2.RequestBuffering {
		return false
	}

	return true
}

type proxy struct {
	backendResolver resolver.DefaultBackend
}

// NewParser creates a new reverse proxy configuration annotation parser
func NewParser(br resolver.DefaultBackend) parser.IngressAnnotation {
	return proxy{br}
}

// ParseAnnotations parses the annotations contained in the ingress
// rule used to configure upstream check parameters
func (a proxy) Parse(ing *networking.Ingress) (interface{}, error) {
	defBackend := a.backendResolver.GetDefaultBackend()
	ct, err := parser.GetIntAnnotation(connect, ing)
	if err != nil {
		ct = defBackend.ProxyConnectTimeout
	}

	st, err := parser.GetIntAnnotation(send, ing)
	if err != nil {
		st = defBackend.ProxySendTimeout
	}

	rt, err := parser.GetIntAnnotation(read, ing)
	if err != nil {
		rt = defBackend.ProxyReadTimeout
	}

	bufs, err := parser.GetStringAnnotation(bufferSize, ing)
	if err != nil || bufs == "" {
		bufs = defBackend.ProxyBufferSize
	}

	cp, err := parser.GetStringAnnotation(cookiePath, ing)
	if err != nil || cp == "" {
		cp = defBackend.ProxyCookiePath
	}

	cd, err := parser.GetStringAnnotation(cookieDomain, ing)
	if err != nil || cd == "" {
		cd = defBackend.ProxyCookieDomain
	}

	bs, err := parser.GetStringAnnotation(bodySize, ing)
	if err != nil || bs == "" {
		bs = defBackend.ProxyBodySize
	}
	if !bodySizeRegex.MatchString(bs) {
		if bs != "unlimited" {
			glog.Warningf("ignoring invalid body size '%v' on %v/%v", bs, ing.Namespace, ing.Name)
		}
		bs = ""
	}

	nu, err := parser.GetStringAnnotation(nextUpstream, ing)
	if err != nil || nu == "" {
		nu = defBackend.ProxyNextUpstream
	}

	pp, err := parser.GetStringAnnotation(passParams, ing)
	if err != nil || pp == "" {
		pp = defBackend.ProxyPassParams
	}

	rb, err := parser.GetStringAnnotation(requestBuffering, ing)
	if err != nil || rb == "" {
		rb = defBackend.ProxyRequestBuffering
	}

	return &Configuration{bs, ct, st, rt, bufs, cd, cp, nu, pp, rb}, nil
}

// IsValidProxyBodySize return true if s is a valid proxy body size string
func IsValidProxyBodySize(s string) bool {
	return bodySizeRegex.MatchString(s)
}
