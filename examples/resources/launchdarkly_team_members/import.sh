# Import a group of existing members as one batch, using a comma-separated list
# of their 24 character member IDs. Every ID is looked up and the members map is
# keyed by each member's email address; the import fails if any ID is unknown.
terraform import launchdarkly_team_members.payments_team '5f05565b48be0b441fb63020,5f05565b48be0b441fb63021'
