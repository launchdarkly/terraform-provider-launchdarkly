# sanitize.jq — replace LD-side identifiers in captured state with the
# deterministic placeholders listed in safe-placeholders.txt. Run by
# capture.sh between `terraform apply` and the safety scan.
#
# Each pass walks the state tree and rewrites well-known fields:
#
#   - launchdarkly_access_token.token / display_token: real secrets get
#     mapped to a non-UUID placeholder so GitHub secret scanning doesn't
#     flag the fixture (and scan.sh's api-<uuid> pattern doesn't fire).
#   - launchdarkly_relay_proxy_configuration.full_key / display_key:
#     same approach with FIXTURE_REDACTED_RELAY_KEY.
#   - launchdarkly_team_member.email / _id: opaque IDs and emails get
#     replaced so the fixture is captured-once / replay-anywhere.
#
# Adding new resource shapes? Append a rewrite stanza below and bump
# safe-placeholders.txt only when the placeholder cannot be derived
# from an existing entry.

def sanitize_attrs(attrs):
  attrs
  # Replace LD access-token secrets with a clearly-synthetic, non-UUID
  # placeholder so GitHub secret scanning doesn't flag the fixture and
  # so scan.sh's `api-<uuid>` pattern doesn't fire either.
  | if has("token") and (.token | type) == "string" and (.token | startswith("api-")) then
      .token = "FIXTURE_REDACTED_TOKEN"
    else . end
  | if has("display_token") and (.display_token | type) == "string" then
      .display_token = "RDCT"
    else . end
  | if has("full_key") and (.full_key | type) == "string" and (.full_key | startswith("rel-")) then
      .full_key = "FIXTURE_REDACTED_RELAY_KEY"
    else . end
  | if has("display_key") and (.display_key | type) == "string" then
      .display_key = "RDCT"
    else . end
  | if has("maintainer_id") and (.maintainer_id | type) == "string" and (.maintainer_id | test("^[a-f0-9]{24}$")) then
      .maintainer_id = "000000000000000000000000"
    else . end
  | if has("_id") and (._id | type) == "string" and (._id | test("^[a-f0-9]{24}$")) then
      ._id = "000000000000000000000000"
    else . end
  | if has("email") and (.email | type) == "string" and (.email | test("@")) then
      .email = "fixture-team-member-PLACEHOLDER@example.invalid"
    else . end
  ;

.resources |= map(
  .instances |= map(
    if has("attributes") then
      .attributes |= sanitize_attrs(.)
    else . end
  )
)
