# Any team referenced in team_keys must exist before the members are invited.
# Referencing the team resource rather than repeating its key as a string is
# what tells Terraform to create the team first.
resource "launchdarkly_team" "payments" {
  key  = "payments"
  name = "Payments"
}

# Invites up to 50 members, with their team assignments, in a single API call.
# The members map is keyed by lowercase email address.
resource "launchdarkly_team_members" "payments_team" {
  members = {
    "alice@example.com" = {
      first_name = "Alice"
      last_name  = "Smith"
      role       = "writer"
      team_keys  = [launchdarkly_team.payments.key]
    }
    "bob@example.com" = {
      custom_roles = ["payments-developer"]
      team_keys    = [launchdarkly_team.payments.key]
    }
  }

  # Both of these default to the values shown; they are spelled out here
  # because they govern what happens to real people's accounts.
  #
  # deletion_protection blocks a destroy, and any update that would remove
  # every member, until you disable it in its own apply.
  deletion_protection = true
  # adopt_existing = false makes an apply fail if one of these emails already
  # belongs to a member of the account, rather than quietly taking ownership of
  # someone this resource would later delete.
  adopt_existing = false

  # Prevents Terraform from even planning a destroy of this resource.
  lifecycle {
    prevent_destroy = true
  }
}

# Teams larger than 50 members need more than one resource. Chunk the map and
# declare one launchdarkly_team_members resource per chunk, for example with a
# module that takes a slice of the email-keyed map.
