package launchdarkly

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func roleAttributesSchema(isDataSource bool) *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeSet,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				KEY: {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The key / name of your role attribute. In the example `$${roleAttribute/testAttribute}`, the key is `testAttribute`.",
				},
				VALUES: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Required:    true,
					Description: "A list of values for your role attribute. For example, if your policy statement defines the resource `\"proj/$${roleAttribute/testAttribute}\"`, the values would be the keys of the projects you wanted to assign access to.",
				},
			},
		},
		Optional:    true,
		Computed:    isDataSource,
		Description: "A role attributes block. One block must be defined per role attribute. The key is the role attribute key and the value is a string array of resource keys that apply.",
	}
}

func roleAttributesFromResourceData(rawRoleAttributes []interface{}) *map[string][]string {
	if len(rawRoleAttributes) == 0 {
		return nil
	}
	roleAttributes := make(map[string][]string)
	for _, attribute := range rawRoleAttributes {
		roleAttribute := attribute.(map[string]interface{})
		key := roleAttribute[KEY].(string)
		rawValues := roleAttribute[VALUES].([]interface{})
		roleAttributes[key] = make([]string, 0, len(rawValues))
		for _, v := range rawValues {
			roleAttributes[key] = append(roleAttributes[key], v.(string))
		}
	}
	return &roleAttributes
}

func roleAttributesToResourceData(roleAttributes *map[string][]string) *[]interface{} {
	if roleAttributes == nil {
		return nil
	}
	resourceData := make([]interface{}, 0, len(*roleAttributes))
	for key, values := range *roleAttributes {
		rawValues := make([]interface{}, 0, len(values))
		for _, v := range values {
			rawValues = append(rawValues, v)
		}
		resourceData = append(resourceData, map[string]interface{}{
			KEY:    key,
			VALUES: rawValues,
		})
	}
	return &resourceData
}

func getRoleAttributePatches(d *schema.ResourceData) []ldapi.PatchOperation {
	var patch []ldapi.PatchOperation
	if o, n := d.GetChange(ROLE_ATTRIBUTES); o != n {
		new := roleAttributesFromResourceData(d.Get(ROLE_ATTRIBUTES).(*schema.Set).List())
		if new != nil {
			patch = append(patch, patchReplace("/roleAttributes", new))
		} else {
			patch = append(patch, patchReplace("/roleAttributes", make(map[string][]string)))
		}
	}
	return patch
}
