package main

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/golang-jwt/jwt/v4"
)

// This is a modified version of the code provided in the follow blog post and
// GitHub repository:
// - https://developer.okta.com/blog/2021/01/04/offline-jwt-validation-with-go
// - https://github.com/oktadev/okta-offline-jwt-validation-example/tree/a61cc73bf893686c1efe67ce86448047205826bc
//
// Copyright 2019 Okta, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// GetOIDCPublicKeys return the public keys of the currently set WHARF_HTTP_OIDC_KeysURL.
func GetOIDCPublicKeys(keysURL string) (map[string]*rsa.PublicKey, error) {
	rsaKeys := make(map[string]*rsa.PublicKey)
	resp, err := http.Get(keysURL)
	if err != nil {
		return nil, fmt.Errorf("http GET keys URL: %w", err)
		//log.Error().WithError(err).Message("Could not fetch from KeysURL.")
	}
	var body struct {
		Keys []struct {
			KeyID  string `json:"kid"`
			Number string `json:"n"`
		} `json:"keys"`
	}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return nil, fmt.Errorf("decode keys payload: %w", err)
		//log.Error().WithError(err).Message("Failed to decode login JWT Keys.")
	}
	log.Debug().Message("Updating keys for oidc.")
	rsaExponent := 65537
	for _, key := range body.Keys {
		kid := key.KeyID
		rsakey := new(rsa.PublicKey)
		number, err := base64.RawURLEncoding.DecodeString(key.Number)
		if err != nil {
			return nil, fmt.Errorf("decode JWT 'n' field: %w", err)
		}
		rsakey.N = new(big.Int).SetBytes(number)
		rsakey.E = rsaExponent
		rsaKeys[kid] = rsakey
	}
	return rsaKeys, nil
}

func NewOIDCMiddleware(rsaKeys map[string]*rsa.PublicKey, config OIDCConfig) *OIDCMiddleware {
	return &OIDCMiddleware{
		rsaKeys: rsaKeys,
		config:  config,
	}
}

type OIDCMiddleware struct {
	rsaKeys map[string]*rsa.PublicKey
	config  OIDCConfig
}

// VerifyTokenMiddleware is a gin middleware function that enforces validity of the access bearer token on every
// request. This uses the environment vars WHARF_HTTP_OIDC_IssuerURL and WHARF_HTTP_OIDC_AudienceURL as limiters
// that control the variety of tokens that pass validation.
func (m *OIDCMiddleware) VerifyTokenMiddleware(ginContext *gin.Context) {
	if m.rsaKeys == nil {
		log.Warn().Message("RsaKeys for OIDC have not been set (http:500).")
		ginContext.AbortWithStatus(http.StatusInternalServerError)
	}
	isValid := false
	errorMessage := ""
	tokenString := ginContext.Request.Header.Get("Authorization")
	if !strings.HasPrefix(tokenString, "Bearer ") {
		ginContext.AbortWithStatus(http.StatusUnauthorized)
	}
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return m.rsaKeys[token.Header["kid"].(string)], nil
	})
	if err != nil {
		errorMessage = err.Error()
	} else if !token.Valid {
		errorMessage = "Invalid access bearer token."
	} else if token.Header["alg"] == nil {
		errorMessage = "Missing 'alg' field in authorization JWT header."
	} else if token.Claims.(jwt.MapClaims)["aud"] != m.config.AudienceURL {
		errorMessage = "Invalid 'aud' field in authorization JWT header."
	} else if !strings.Contains(token.Claims.(jwt.MapClaims)["iss"].(string), m.config.IssuerURL) {
		errorMessage = "Invalid 'iss' field in authorization JWT header."
	} else {
		isValid = true
	}
	if !isValid {
		ginContext.String(http.StatusForbidden, errorMessage)
		ginContext.AbortWithStatus(http.StatusUnauthorized)
	}
}

// SubscribeToKeyURLUpdates ensures new keys are fetched as necessary.
// As a standard OIDC login provider keys should be checked for updates ever 1 day 1 hour.
func (m *OIDCMiddleware) SubscribeToKeyURLUpdates() {
	fetchOidcKeysTicker := time.NewTicker(m.config.UpdateInterval)
	log.Debug().WithDuration("interval", m.config.UpdateInterval).
		Message("Subscribing to OIDC public keys rotation via periodic check timer.")
	go func() {
		for {
			<-fetchOidcKeysTicker.C
			m.UpdateOIDCPublicKeys()
		}
	}()
}

func (m *OIDCMiddleware) UpdateOIDCPublicKeys() {
	newKeys, err := GetOIDCPublicKeys(m.config.KeysURL)
	if err != nil {
		log.Warn().WithError(err).
			WithDuration("interval", m.config.UpdateInterval).
			Message("Failed to update OIDC public keys.")
	} else {
		m.rsaKeys = newKeys
		log.Info().
			WithDuration("interval", m.config.UpdateInterval).
			Message("Successfully updated OIDC public keys.")
	}
}
