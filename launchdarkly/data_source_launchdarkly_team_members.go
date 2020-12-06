package launchdarkly

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func dataSourceTeamMembers() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceTeamMembersRead,

		Schema: map[string]*schema.Schema{
			EMAILS: {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			IGNORE_MISSING: {
				Type: 	  schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			TEAM_MEMBERS: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						EMAIL: {
							Type:     schema.TypeString,
							Computed: true,
						},
						FIRST_NAME: {
							Type:     schema.TypeString,
							Computed: true,
						},
						LAST_NAME: {
							Type:     schema.TypeString,
							Computed: true,
						},
						ROLE: {
							Type:     schema.TypeString,
							Computed: true,
						},
						CUSTOM_ROLES: {
							Type:     schema.TypeSet,
							Set:      schema.HashString,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceTeamMembersRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	var members []*ldapi.Member
	expectedCount := 0
	ignoreMissing := d.Get(IGNORE_MISSING).(bool)

	if emails, ok := d.Get(EMAILS).([]interface{}); ok && len(emails) > 0 {
		expectedCount = len(emails)
		for _, memberEmail := range emails {
			member, err := getTeamMemberByEmail(client, fmt.Sprintf("%s", memberEmail))
			if err != nil {
				if ignoreMissing {
					continue
				}
				return err
			}
			members = append(members, member)
		}
	}

	if !ignoreMissing && len(members) != expectedCount {
		return fmt.Errorf("unexpected number of users returned (%d != %d)", len(members), expectedCount)
	}

	ids := make([]string, 0, len(members))
	memberList := make([]map[string]interface{}, 0, len(members))
	for _, m := range members {
		member := make(map[string]interface{})
		member[EMAIL] = m.Email
		member[FIRST_NAME] = m.FirstName
		member[LAST_NAME] = m.LastName
		member[ROLE] = m.Role
		member[CUSTOM_ROLES] = m.CustomRoles
		memberList = append(memberList, member)
		ids = append(ids, m.Id)
	}

	h := sha1.New()
	if _, err := h.Write([]byte(strings.Join(ids, "-"))); err != nil {
		return fmt.Errorf("unable to compute hash for IDs: %v", err)
	}
	d.SetId("team_members#" + base64.URLEncoding.EncodeToString(h.Sum(nil)))

	_ = d.Set(TEAM_MEMBERS, memberList)

	return nil
}
