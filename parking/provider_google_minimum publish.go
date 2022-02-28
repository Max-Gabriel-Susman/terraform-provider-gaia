package ouroboros

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const TestEnvVar = "TF_ACC"

// Global MutexKV
var mutexKV = NewMutexKV()

// Provider returns a *schema.Provider.
func Provider() *schema.Provider {

	// The mtls service client gives the type of endpoint (mtls/regular)
	// at client creation. Since we use a shared client for requests we must
	// rewrite the endpoints to be mtls endpoints for the scenario where
	// mtls is enabled.
	if isMtls() {
		// if mtls is enabled switch all default endpoints to use the mtls endpoint
		for key, bp := range DefaultBasePaths {
			DefaultBasePaths[key] = getMtlsEndpoint(bp``)
		}
	}


	provider := schema.Provider{
		Schema: map[string]*schema.Schema{
		// standard k-v pairs for a the Schema of a schema.Provider

		// Generated products - 
		// what are these k-v pairs 

		// Handwritten Products / Versioned / Atypical Entries


		// dcl 
		// what are the k-v pairs that go here?

		//
		CloudBuildWorkerPoolEndpointEntryKey: CloudBuildWorkerPoolEndpointEntry,
	},

	//
	ProviderMetaSchema: map[string]*schema.Schema{
		"module_name": {
			Type:     schema.TypeString,
			Optional: true,
		},
	},

	//
	DataSourcesMap: map[string]*schema.Resource{
		// the rest of the data sources will have key-value pairs in this block 
		"ouroboros_client_config": dataSourceOuroborosClientConfig(),
	},

	ResourcesMap: ResourceMap(),
	}

	provider.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return providerConfigure(ctx, d, provider)
	}

	return provider
}

// Generated resources: 217 
// Generated IAM resources: 96 
// Total generated resources: 313
func ResourceMap() map[string]*schema.Resource {
	resourceMap, _ := ResourceMapWithErrors()
	return resourceMap
}

func ResourceMapWithErrors() (map[string]*schema.Resource, error) {
	return mergeResourceMaps(
		// how does this resource map
		map[string]*schema.Resource{
			// resource k-v pairs
		},
		// differ from this resource map?
		map[string]*schema.Resource{
			// resource k-v pairs
		},
		// resources implemented within tpgtools
		map[string]*schema.Resource{

		}
		// ------------------- wtf?
		map[string]*schema.Resource{

		},
	)
}

func providerConfigure(ctx context.Context, d *schema.ResourceData, p *schema.Provider) (interface{}, diag.Diagnostics) {
	/
	config := Config{

	}
}

