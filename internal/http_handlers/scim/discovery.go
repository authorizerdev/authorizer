package scim

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// serviceProviderConfig advertises which SCIM features this server supports so
// an IdP knows to use PATCH (deprovisioning), no bulk, single-term filtering
// (eq/ne/co/sw/pr over the core User attributes), and bearer auth (RFC 7644 §5).
func (h *Handler) serviceProviderConfig(c *gin.Context) {
	cfg := gin.H{
		"schemas":          []string{schemaSPConfig},
		"documentationUri": "https://docs.authorizer.dev",
		"patch":            gin.H{"supported": true},
		"bulk":             gin.H{"supported": false, "maxOperations": 0, "maxPayloadSize": 0},
		"filter":           gin.H{"supported": true, "maxResults": 1000},
		"changePassword":   gin.H{"supported": false},
		"sort":             gin.H{"supported": false},
		"etag":             gin.H{"supported": false},
		"authenticationSchemes": []gin.H{{
			"type":        "oauthbearertoken",
			"name":        "OAuth Bearer Token",
			"description": "Authentication via the per-organization SCIM bearer token.",
			"primary":     true,
		}},
		"meta": gin.H{"resourceType": "ServiceProviderConfig"},
	}
	c.Header("Content-Type", contentType)
	c.JSON(http.StatusOK, cfg)
}

// resourceTypes lists the provisionable resource types: User and Group. Group
// membership is stored as org-namespaced FGA tuples (RFC 7643 §4.1.2).
func (h *Handler) resourceTypes(c *gin.Context) {
	user := gin.H{
		"schemas":     []string{schemaRestype},
		"id":          "User",
		"name":        "User",
		"endpoint":    "/Users",
		"description": "User Account",
		"schema":      schemaUser,
		"meta":        gin.H{"resourceType": "ResourceType"},
	}
	group := gin.H{
		"schemas":     []string{schemaRestype},
		"id":          "Group",
		"name":        "Group",
		"endpoint":    "/Groups",
		"description": "Group",
		"schema":      schemaGroup,
		"meta":        gin.H{"resourceType": "ResourceType"},
	}
	resp := gin.H{
		"schemas":      []string{schemaListResp},
		"totalResults": 2,
		"startIndex":   1,
		"itemsPerPage": 2,
		"Resources":    []gin.H{user, group},
	}
	c.Header("Content-Type", contentType)
	c.JSON(http.StatusOK, resp)
}

// schemas describes the User attributes this server understands. Kept to the
// subset actually mapped (userName, name, emails, externalId, active).
func (h *Handler) schemas(c *gin.Context) {
	attr := func(name, typ string, required bool) gin.H {
		return gin.H{
			"name": name, "type": typ, "required": required,
			"multiValued": false, "mutability": "readWrite",
			"returned": "default", "uniqueness": "none",
		}
	}
	immutableAttr := func(name, typ string) gin.H {
		return gin.H{
			"name": name, "type": typ, "required": false,
			"multiValued": false, "mutability": "immutable",
			"returned": "default", "uniqueness": "none",
		}
	}
	userSchema := gin.H{
		"schemas":     []string{schemaSchema},
		"id":          schemaUser,
		"name":        "User",
		"description": "User Account",
		"attributes": []gin.H{
			attr("userName", "string", true),
			attr("externalId", "string", false),
			attr("active", "boolean", false),
			{
				"name": "name", "type": "complex", "required": false,
				"multiValued": false, "mutability": "readWrite",
				"returned": "default", "uniqueness": "none",
				"subAttributes": []gin.H{
					attr("givenName", "string", false),
					attr("familyName", "string", false),
				},
			},
			{
				"name": "emails", "type": "complex", "required": false,
				"multiValued": true, "mutability": "readWrite",
				"returned": "default", "uniqueness": "none",
				"subAttributes": []gin.H{
					attr("value", "string", false),
					attr("primary", "boolean", false),
				},
			},
		},
		"meta": gin.H{"resourceType": "Schema"},
	}
	groupSchema := gin.H{
		"schemas":     []string{schemaSchema},
		"id":          schemaGroup,
		"name":        "Group",
		"description": "Group",
		"attributes": []gin.H{
			attr("displayName", "string", true),
			attr("externalId", "string", false),
			{
				"name": "members", "type": "complex", "required": false,
				"multiValued": true, "mutability": "readWrite",
				"returned": "default", "uniqueness": "none",
				// RFC 7643 §4.2: the members sub-attributes are immutable.
				"subAttributes": []gin.H{
					immutableAttr("value", "string"),
					immutableAttr("$ref", "reference"),
					immutableAttr("type", "string"),
					immutableAttr("display", "string"),
				},
			},
		},
		"meta": gin.H{"resourceType": "Schema"},
	}
	resp := gin.H{
		"schemas":      []string{schemaListResp},
		"totalResults": 2,
		"startIndex":   1,
		"itemsPerPage": 2,
		"Resources":    []gin.H{userSchema, groupSchema},
	}
	c.Header("Content-Type", contentType)
	c.JSON(http.StatusOK, resp)
}
