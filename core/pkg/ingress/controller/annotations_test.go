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

package controller

import (
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/util/intstr"

	"k8s.io/ingress/core/pkg/ingress/defaults"
	"k8s.io/ingress/core/pkg/ingress/resolver"
)

const (
	annotation_secureUpstream = "ingress.kubernetes.io/secure-backends"
	annotation_upsMaxFails    = "ingress.kubernetes.io/upstream-max-fails"
	annotation_upsFailTimeout = "ingress.kubernetes.io/upstream-fail-timeout"
	annotation_passthrough    = "ingress.kubernetes.io/ssl-passthrough"
)

type mockCfg struct {
}

func (m mockCfg) GetDefaultBackend() defaults.Backend {
	return defaults.Backend{}
}

func (m mockCfg) GetSecret(string) (*api.Secret, error) {
	return nil, nil
}

func (m mockCfg) GetAuthCertificate(string) (*resolver.AuthSSLCert, error) {
	return nil, nil
}

func TestAnnotationExtractor(t *testing.T) {
	ec := newAnnotationExtractor(mockCfg{})
	ing := buildIngress()

	m := ec.Extract(ing)
	// the map at least should contains HealthCheck and Proxy information (defaults)
	if _, ok := m["HealthCheck"]; !ok {
		t.Error("expected HealthCheck annotation")
	}
	if _, ok := m["Proxy"]; !ok {
		t.Error("expected Proxy annotation")
	}
}

func buildIngress() *extensions.Ingress {
	defaultBackend := extensions.IngressBackend{
		ServiceName: "default-backend",
		ServicePort: intstr.FromInt(80),
	}

	return &extensions.Ingress{
		ObjectMeta: api.ObjectMeta{
			Name:      "foo",
			Namespace: api.NamespaceDefault,
		},
		Spec: extensions.IngressSpec{
			Backend: &extensions.IngressBackend{
				ServiceName: "default-backend",
				ServicePort: intstr.FromInt(80),
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

func TestSecureUpstream(t *testing.T) {
	ec := newAnnotationExtractor(mockCfg{})
	ing := buildIngress()

	fooAnns := []struct {
		annotations map[string]string
		er          bool
	}{
		{map[string]string{annotation_secureUpstream: "true"}, true},
		{map[string]string{annotation_secureUpstream: "false"}, false},
		{map[string]string{annotation_secureUpstream + "_no": "true"}, false},
		{map[string]string{}, false},
		{nil, false},
	}

	for _, foo := range fooAnns {
		ing.SetAnnotations(foo.annotations)
		r := ec.SecureUpstream(ing)
		if r != foo.er {
			t.Errorf("Returned %v but expected %v", r, foo.er)
		}
	}
}

func TestHealthCheck(t *testing.T) {
	ec := newAnnotationExtractor(mockCfg{})
	ing := buildIngress()

	fooAnns := []struct {
		annotations map[string]string
		eumf        int
		euft        int
	}{
		{map[string]string{annotation_upsMaxFails: "3", annotation_upsFailTimeout: "10"}, 3, 10},
		{map[string]string{annotation_upsMaxFails: "3"}, 3, 0},
		{map[string]string{annotation_upsFailTimeout: "10"}, 0, 10},
		{map[string]string{}, 0, 0},
		{nil, 0, 0},
	}

	for _, foo := range fooAnns {
		ing.SetAnnotations(foo.annotations)
		r := ec.HealthCheck(ing)
		if r == nil {
			t.Errorf("Returned nil but expected a healthcheck.Upstream")
			continue
		}

		if r.FailTimeout != foo.euft {
			t.Errorf("Returned %d but expected %d for FailTimeout", r.FailTimeout, foo.euft)
		}

		if r.MaxFails != foo.eumf {
			t.Errorf("Returned %d but expected %d for MaxFails", r.MaxFails, foo.eumf)
		}
	}
}

func TestSSLPassthrough(t *testing.T) {
	ec := newAnnotationExtractor(mockCfg{})
	ing := buildIngress()

	fooAnns := []struct {
		annotations map[string]string
		er          bool
	}{
		{map[string]string{annotation_passthrough: "true"}, true},
		{map[string]string{annotation_passthrough: "false"}, false},
		{map[string]string{annotation_passthrough + "_no": "true"}, false},
		{map[string]string{}, false},
		{nil, false},
	}

	for _, foo := range fooAnns {
		ing.SetAnnotations(foo.annotations)
		r := ec.SSLPassthrough(ing)
		if r != foo.er {
			t.Errorf("Returned %v but expected %v", r, foo.er)
		}
	}
}
