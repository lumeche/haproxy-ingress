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

package authtls

import (
	"github.com/pkg/errors"
	extensions "k8s.io/api/extensions/v1beta1"

	"github.com/lumeche/haproxy-ingress/pkg/common/ingress/annotations/parser"
	ing_errors "github.com/lumeche/haproxy-ingress/pkg/common/ingress/errors"
	"github.com/lumeche/haproxy-ingress/pkg/common/ingress/resolver"
	"github.com/lumeche/haproxy-ingress/pkg/common/k8s"
	"regexp"
)

const (
	// name of the secret
	annotationAuthTLSSecret    = "ingress.kubernetes.io/auth-tls-secret"
	annotationCertHeader       = "ingress.kubernetes.io/auth-tls-cert-header"
	annotationAuthVerifyClient = "ingress.kubernetes.io/auth-tls-verify-client"
	annotationAuthTLSDepth     = "ingress.kubernetes.io/auth-tls-verify-depth"
	annotationAuthTLSErrorPage = "ingress.kubernetes.io/auth-tls-error-page"
	defaultAuthTLSDepth        = 1
	defaultAuthVerifyClient    = "on"
)

var (
	authVerifyClientRegex = regexp.MustCompile(`on|off|optional|optional_no_ca`)
)

// AuthSSLConfig contains the AuthSSLCert used for muthual autentication
// and the configured ValidationDepth
type AuthSSLConfig struct {
	resolver.AuthSSLCert
	CertHeader      bool   `json:"certHeader"`
	VerifyClient    string `json:"verify_client"`
	ValidationDepth int    `json:"validationDepth"`
	ErrorPage       string `json:"errorPage"`
}

// Equal tests for equality between two AuthSSLConfig types
func (assl1 *AuthSSLConfig) Equal(assl2 *AuthSSLConfig) bool {
	if assl1 == assl2 {
		return true
	}
	if assl1 == nil || assl2 == nil {
		return false
	}
	if assl1.CertHeader != assl2.CertHeader {
		return false
	}
	if !(&assl1.AuthSSLCert).Equal(&assl2.AuthSSLCert) {
		return false
	}
	if assl1.VerifyClient != assl2.VerifyClient {
		return false
	}
	if assl1.ValidationDepth != assl2.ValidationDepth {
		return false
	}
	if assl1.ErrorPage != assl2.ErrorPage {
		return false
	}
	return true
}

// NewParser creates a new TLS authentication annotation parser
func NewParser(resolver resolver.AuthCertificate) parser.IngressAnnotation {
	return authTLS{resolver}
}

type authTLS struct {
	certResolver resolver.AuthCertificate
}

// Parse parses the annotations contained in the ingress
// rule used to use a Certificate as authentication method
func (a authTLS) Parse(ing *extensions.Ingress) (interface{}, error) {

	tlsauthsecret, err := parser.GetStringAnnotation(annotationAuthTLSSecret, ing)
	if err != nil {
		return &AuthSSLConfig{}, err
	}

	if tlsauthsecret == "" {
		return &AuthSSLConfig{}, ing_errors.NewLocationDenied("an empty string is not a valid secret name")
	}

	_, _, err = k8s.ParseNameNS(tlsauthsecret)
	if err != nil {
		return &AuthSSLConfig{}, ing_errors.NewLocationDenied(err.Error())
	}

	certHeader, err := parser.GetBoolAnnotation(annotationCertHeader, ing)
	if err != nil {
		certHeader = false
	}

	tlsVerifyClient, err := parser.GetStringAnnotation(annotationAuthVerifyClient, ing)
	if err != nil || !authVerifyClientRegex.MatchString(tlsVerifyClient) {
		tlsVerifyClient = defaultAuthVerifyClient
	}

	tlsdepth, err := parser.GetIntAnnotation(annotationAuthTLSDepth, ing)
	if err != nil || tlsdepth == 0 {
		tlsdepth = defaultAuthTLSDepth
	}

	authCert, err := a.certResolver.GetAuthCertificate(tlsauthsecret)
	if err != nil {
		return &AuthSSLConfig{}, ing_errors.LocationDenied{
			Reason: errors.Wrap(err, "error obtaining certificate"),
		}
	}

	errorpage, err := parser.GetStringAnnotation(annotationAuthTLSErrorPage, ing)
	if err != nil || errorpage == "" {
		errorpage = ""
	}

	return &AuthSSLConfig{
		AuthSSLCert:     *authCert,
		CertHeader:      certHeader,
		VerifyClient:    tlsVerifyClient,
		ValidationDepth: tlsdepth,
		ErrorPage:       errorpage,
	}, nil
}
