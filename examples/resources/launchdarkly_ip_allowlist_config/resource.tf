# IP allowlists are an Enterprise feature and use a beta API. There is one IP
# allowlist configuration per account, so define only a single instance of this resource.
resource "launchdarkly_ip_allowlist_config" "example" {
  session_allowlist_enabled = true
  scoped_allowlist_enabled  = true
}
