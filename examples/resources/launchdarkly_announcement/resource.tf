resource "launchdarkly_announcement" "scheduled_maintenance" {
  title          = "Scheduled maintenance"
  message        = "LaunchDarkly will be undergoing **scheduled maintenance** on January 1st. The app may be briefly unavailable."
  severity       = "warning"
  is_dismissible = true
  start_time     = 1893456000000 # Unix timestamp in milliseconds (2030-01-01T00:00:00Z)
  end_time       = 1924992000000 # Unix timestamp in milliseconds (2031-01-01T00:00:00Z)
}
