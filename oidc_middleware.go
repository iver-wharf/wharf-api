package main

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

// Largely taken from https://developer.okta.com/blog/2021/01/04/offline-jwt-validation-with-go

type oidcService struct {
	rsakeys *map[string]*rsa.PublicKey
}

func (oidc oidcService) GetPublicKeys(config OICDConfig) {
	var body map[string]interface{}
	resp, err := http.Get(config.KeysURL)
	if err != nil {
		log.Error().WithError(err).Message("Could not fetch from KeysURL.")
	}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		log.Error().WithError(err).Message("Failed to decode login JWT Keys.")
	}
	for _, bodykey := range body["keys"].([]interface{}) {
		key := bodykey.(map[string]interface{})
		kid := key["kid"].(string)
		rsakey := new(rsa.PublicKey)
		number, _ := base64.RawURLEncoding.DecodeString(key["n"].(string))
		rsakey.N = new(big.Int).SetBytes(number)
		rsakey.E = 65537
		(*oidc.rsakeys)[kid] = rsakey
	}
}

// As a standard OICD login provider keys should be checked for updates ever 1 day 1 hour.
func (oidc oidcService) SubscribeToKeyURLUpdates() {
	// TODO Implement
	panic("If you see this then you should implement this logic.")
}

func (oidc oidcService) VerifyTokenMiddleware(config OICDConfig) gin.HandlerFunc {
	return func (ginContext *gin.Context) {
		// middleware
		isValid := false
		errorMessage := ""
		tokenString := ginContext.Request.Header.Get("Authorization")
		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = strings.TrimPrefix(tokenString, "Bearer ")
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return (*oidc.rsakeys)[token.Header["kid"].(string)], nil
			})
			if err != nil {
				errorMessage = err.Error()
			} else if !token.Valid {
				errorMessage = "Invalid token"
			} else if token.Header["alg"] == nil {
				errorMessage = "alg must be defined"
			} else if token.Claims.(jwt.MapClaims)["aud"] != config.AudienceURL {
				errorMessage = "Invalid aud"
			} else if !strings.Contains(token.Claims.(jwt.MapClaims)["iss"].(string), config.IssuerURL) {
				errorMessage = "Invalid iss"
			} else {
				isValid = true
			}
			if !isValid {
				ginContext.String(http.StatusForbidden, errorMessage)
				ginContext.AbortWithStatus(http.StatusUnauthorized)
			}
		} else {
			ginContext.AbortWithStatus(http.StatusUnauthorized)
		}
	}
}
