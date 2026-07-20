resource "launchdarkly_oauth_client" "internal_dashboard" {
  name         = "Internal dashboard"
  redirect_uri = "https://dashboard.example.com/oauth/callback"
  description  = "OAuth 2.0 client for our internal analytics dashboard"
}
