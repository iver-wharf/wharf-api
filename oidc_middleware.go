package main

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
)
// Largely taken from https://developer.okta.com/blog/2021/01/04/offline-jwt-validation-with-go

func GetOidcPublicKeys(config OICDConfig) *map[string]*rsa.PublicKey {
	rsakeys := make(map[string]*rsa.PublicKey)
	var body map[string]interface{}
	resp, err := http.Get(config.KeysURL)
	if err != nil {
		log.Error().WithError(err).Message("Could not fetch from KeysURL.")
	}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		log.Error().WithError(err).Message("Failed to decode login JWT Keys.")
	}
	log.Debug().Message("Updating keys for oidc.")
	for _, bodykey := range body["keys"].([]interface{}) {
		key := bodykey.(map[string]interface{})
		kid := key["kid"].(string)
		rsakey := new(rsa.PublicKey)
		number, _ := base64.RawURLEncoding.DecodeString(key["n"].(string))
		rsakey.N = new(big.Int).SetBytes(number)
		rsakey.E = 65537
		rsakeys[kid] = rsakey
	}
	return &rsakeys
}

func VerifyTokenMiddleware(config OICDConfig, rsakeys *map[string]*rsa.PublicKey) gin.HandlerFunc {
	return func (ginContext *gin.Context) {
		if *rsakeys == nil {
			log.Warn().Message("RsaKeys for OIDC have not been set (http:500).")
			ginContext.AbortWithStatus(http.StatusInternalServerError)
		}
		isValid := false
		errorMessage := ""
		tokenString := ginContext.Request.Header.Get("Authorization")
		if !strings.HasPrefix(tokenString, "Bearer ") {
			ginContext.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return (*rsakeys)[token.Header["kid"].(string)], nil
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
	}
}

// As a standard OICD login provider keys should be checked for updates ever 1 day 1 hour.
func SubscribeToKeyURLUpdates(config OICDConfig, rsakeys *map[string]*rsa.PublicKey) {
	fetchOidcKeysTicker := time.NewTicker(time.Hour * 25)
	go func() {
		for {
			<-fetchOidcKeysTicker.C
			rsakeys = GetOidcPublicKeys(config)
		}
	}()
}
