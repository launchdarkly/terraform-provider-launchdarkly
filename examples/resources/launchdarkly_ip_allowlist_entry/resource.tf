# IP allowlist entries are an Enterprise feature and use a beta API.
# The ip_address may be a single address or a CIDR block. Changing it forces a new entry.
resource "launchdarkly_ip_allowlist_entry" "office" {
  ip_address  = "203.0.113.0/24"
  description = "Corporate office network"
}

resource "launchdarkly_ip_allowlist_entry" "vpn" {
  ip_address  = "198.51.100.42"
  description = "VPN egress IP"
}
