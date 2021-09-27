// Package request contains plain old Go types used in the Gin endpoint handlers
// and Swaggo documentation for the HTTP request models, with Gin- and
// Swaggo-specific Go tags.
package request

// Reference doc about the Go tags:
//  TAG                  SOURCE                   DESCRIPTION
//  json:"foo"           encoding/json            Serializes field with the name "foo"
//  format:"date-time"   swaggo/swag              Swagger format
//  validate:"required"  swaggo/swag              Mark Swagger field as required/non-nullable
//  binding:"required"   go-playground/validator  Gin's Bind will error if nil or zero
//
// go-playground/validator uses the tag "validate" by default, but Gin overrides
// changes that to "binding".

// TokenSearch holds values used in verbatim searches for tokens.
type TokenSearch struct {
	Token    string `json:"token" format:"password"`
	UserName string `json:"userName"`
}

// Token specifies fields when creating a new token.
type Token struct {
	Token      string `json:"token" format:"password" validate:"required"`
	UserName   string `json:"userName" validate:"required"`
	ProviderID uint   `json:"providerId"`
}

// Token specifies fields when adding a new branch to a project.
type Branch struct {
	ProjectID uint   `json:"projectId" validate:"required"`
	Name      string `json:"name" validate:"required"`
	Default   bool   `json:"default"`
	TokenID   uint   `json:"tokenId"`
}
