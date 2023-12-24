// Terraform provider for pfSense.
//
// This provider allows Terraform to manage resources in pfSense by interfacing with its API.
// Supported authentication methods include local authentication, JWT, and token-based authentication.
// Only one form of authentication should be configured per instance.
//
// Usage:
//
// provider "pfsense" {
//     url               = "https://192.168.0.1"
//     user              = "your_username"           // Optional: For local auth.
//     password          = "your_password"           // Optional: For local auth.
//     jwt_token         = "your_jwt_token"          // Optional: For JWT auth.
//     api_client_id     = "your_client_id"          // Optional: For token auth.
//     api_client_token  = "your_client_token"       // Optional: For token auth.
//     skip_tls          = false                     // Optional: Default is false.
//     timeout           = 30                        // Optional: Default is 30 seconds.
// }
//
// Notes:
// - JWTAuthEnabled is inferred from the presence of `jwt_token`.
// - LocalAuthEnabled is inferred from the presence of `user`.
// - TokenAuthEnabled is inferred from the presence of `api_client_id`.
//
// Created by: [Your Name or Alias]
// Date: [Creation Date]
// Target Terraform Version: [X.X.X]
// Target pfSense API Version: [X.X.X]

package pfsense

import (
	"errors"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

// Provider returns a Terraform provider for managing pfSense resources.
func Provider() *schema.Provider {
	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
				Description:  "The url of the target pfsense e.g https://192.168.1.1",
			},
			"user": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Local authentication username.",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Local authentication password.",
				Sensitive:   true,
			},
			"jwt_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "JWT token for authentication.",
				Sensitive:   true,
			},
			"api_client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "API Client ID for token-based authentication.",
			},
			"api_client_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "API Client Token for token-based authentication.",
				Sensitive:   true,
			},
			"allow_insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Skip TLS verification. If not specified, it defaults to true unless the url uses HTTPS.",
			},
			"timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Request timeout duration in seconds.",
				Default:     60,
			},
		},
		ResourcesMap:  map[string]*schema.Resource{},
		ConfigureFunc: providerConfigure,
	}

	resourceFirewallAlias().AddResource(provider)
	resourceDHCPServer().AddResource(provider)
	resourceFirewallRule().AddResource(provider)
	resourceDHCPStaticMapping().AddResource(provider)
	resourceInterface().AddResource(provider)
	resourceInterfaceVLAN().AddResource(provider)

	return provider
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	url := d.Get("url").(string)

	d.Get("allow_insecure")
	allowInsecure := d.Get("allow_insecure").(bool) || strings.HasPrefix(url, "https://")

	c := pfsenseapi.Config{
		Host:    url,
		SkipTLS: allowInsecure,
		Timeout: time.Duration(d.Get("timeout").(int)) * time.Second,
	}

	// Check for JWT auth
	if jwtToken, ok := d.GetOk("jwt_token"); ok {
		c.JWTAuthEnabled = true
		c.JWTToken = jwtToken.(string)
	}

	// Check for local auth
	if user, ok := d.GetOk("user"); ok {
		c.LocalAuthEnabled = true
		c.User = user.(string)

		if password, ok := d.GetOk("password"); !ok {
			return nil, errors.New("password is required when username is provided")
		} else {
			c.Password = password.(string)
		}
	}

	// Check for token auth
	if clientID, ok := d.GetOk("api_client_id"); ok {
		c.TokenAuthEnabled = true
		c.ApiClientID = clientID.(string)

		if clientToken, ok := d.GetOk("api_client_token"); !ok {
			return nil, errors.New("api_client_token is required when api_client_id is provided")
		} else {
			c.ApiClientToken = clientToken.(string)
		}
	}

	// Validate only one form of auth is present
	authCount := 0
	if c.JWTAuthEnabled {
		authCount++
	}
	if c.LocalAuthEnabled {
		authCount++
	}
	if c.TokenAuthEnabled {
		authCount++
	}

	if authCount > 1 {
		return nil, errors.New("only one form of authentication should be provided")
	}

	client := pfsenseapi.NewClient(c)
	return client, nil
}
