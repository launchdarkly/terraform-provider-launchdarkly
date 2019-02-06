package launchdarkly

import (
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/launchdarkly/api-client-go"
)

func stringPtr(v string) *string { return &v }

func stringList(v []interface{}) []string {
	list := make([]string, len(v))
	for i, elem := range v {
		list[i] = elem.(string)
	}
	return list
}

func patchReplace(path string, value interface{}) ldapi.PatchOperation {
	return ldapi.PatchOperation{
		Op:    "replace",
		Path:  path,
		Value: &value,
	}
}

func stringSetFromResourceData(d *schema.ResourceData, key string) []string {
	tags := d.Get(key).(*schema.Set)

	strs := make([]string, tags.Len())
	for i, tag := range tags.List() {
		strs[i] = tag.(string)
	}
	return strs
}

func stringSchemaSetFunc(value interface{}) int {
	return hashcode.String(value.(string))
}

var stringSchemaSet schema.SchemaSetFunc = stringSchemaSetFunc
