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

package ipwhitelist

import (
	"github.com/golang/glog"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/jcmoraisjr/haproxy-ingress/pkg/common/net"
	networking "k8s.io/api/networking/v1"

	"github.com/jcmoraisjr/haproxy-ingress/pkg/common/ingress/annotations/parser"
	ing_errors "github.com/jcmoraisjr/haproxy-ingress/pkg/common/ingress/errors"
	"github.com/jcmoraisjr/haproxy-ingress/pkg/common/ingress/resolver"
)

const (
	whitelist = "ingress.kubernetes.io/whitelist-source-range"
)

// SourceRange returns the CIDR
type SourceRange struct {
	CIDR []string `json:"cidr,omitEmpty"`
}

// Equal tests for equality between two SourceRange types
func (sr1 *SourceRange) Equal(sr2 *SourceRange) bool {
	if sr1 == sr2 {
		return true
	}
	if sr1 == nil || sr2 == nil {
		return false
	}

	if len(sr1.CIDR) != len(sr2.CIDR) {
		return false
	}

	for _, s1l := range sr1.CIDR {
		found := false
		for _, sl2 := range sr2.CIDR {
			if s1l == sl2 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

type ipwhitelist struct {
	backendResolver resolver.DefaultBackend
}

// NewParser creates a new whitelist annotation parser
func NewParser(br resolver.DefaultBackend) parser.IngressAnnotation {
	return ipwhitelist{br}
}

// ParseAnnotations parses the annotations contained in the ingress
// rule used to limit access to certain client addresses or networks.
// Multiple ranges can specified using commas as separator
// e.g. `18.0.0.0/8,56.0.0.0/8`
func (a ipwhitelist) Parse(ing *networking.Ingress) (interface{}, error) {
	defBackend := a.backendResolver.GetDefaultBackend()
	sort.Strings(defBackend.WhitelistSourceRange)

	val, err := parser.GetStringAnnotation(whitelist, ing)
	// A missing annotation is not a problem, just use the default
	if err == ing_errors.ErrMissingAnnotations {
		return &SourceRange{CIDR: defBackend.WhitelistSourceRange}, nil
	}

	values := strings.Split(val, ",")
	ipnets, ips, err := net.ParseIPNets(values...)
	if err != nil {
		if len(ipnets) == 0 && len(ips) == 0 {
			return &SourceRange{CIDR: defBackend.WhitelistSourceRange}, ing_errors.LocationDenied{
				Reason: errors.Wrap(err, "the annotation does not contain a valid IP address or network"),
			}
		}
		glog.Warningf("error parsing %v/%v whitelist-source-range: %v", ing.Namespace, ing.Name, err)
	}

	cidrs := []string{}
	for k := range ipnets {
		cidrs = append(cidrs, k)
	}
	for k := range ips {
		cidrs = append(cidrs, k)
	}

	sort.Strings(cidrs)

	return &SourceRange{cidrs}, nil
}
