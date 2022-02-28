package ouroboros

import (
	"context"

	"github.com/Max-Gabriel-Susman/ouroboros-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema:               map[string]*schema.Schema{},
		ResourcesMap:         map[string]*schema.Resource{},
		DataSourcesMap:       map[string]*schema.Resource{},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	username := d.Get("username").(string)
	password := d.Get("password").(string)

	// Warnings or errors can be collected in a slice type
	var diags diag.Diagnostics

	// diags = append(diags, diag.Diagnostic{
	// 	Severity: diag.Warning,
	// 	Summary:  "Warning Message Summary",
	// 	Detail:   "This is the detailed warning message from providerConfigure",
	// })

	if (username != "") && (password != "") {
		c, err := ouroboros.NewClient(nil, &username, &password)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to create Ouroboros client",
				Detail:   "Unable to auth user for authenticated Ouroboros client",
			})
			return nil, diags
		}
		return c, diags
	}

	c, err := ouroboros.NewClient(nil, nil, nil)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create Ouroboros client",
			Detail:   "Unable to auth user for authenticated Ouroboros client",
		})
		return nil, diags
	}

	return c, diags
}

func validateCredentials(v interface{}, k string) (warnings []string, errors []error) {
	if v == nil || v.(string) == "" {
		return 
	}
	creds := v.(string)
	// if this is a path and we can stat it, assume it's ok (wtf does this mean?)
	if _, err := os.Stat(creds); err == nil {
		return
	}

	if _, err := googleoauth.CredentialsFromJSON(context.Background(), []byte(creds)); err != nil {
		errors = append(errors,
			fmt.Errorf("JSON credentials in %q are not valid: %s", creds, err))``
	}
	return 
}
