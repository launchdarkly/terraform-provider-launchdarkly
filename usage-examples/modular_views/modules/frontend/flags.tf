# Frontend team flags - each flag declares which views it belongs to

resource "launchdarkly_feature_flag" "dark_mode" {
  project_key = var.project_key
  key         = "dark-mode-ui"
  name        = "Dark Mode UI"
  description = "Enable dark mode for the application"
  
  variation_type = "boolean"
  
  client_side_availability {
    using_environment_id = true
    using_mobile_key     = true
  }
  
  # This flag belongs to the frontend team view
  view_keys = [var.team_view_key]
  
  tags = ["frontend", "ui", "dark-mode"]
}

resource "launchdarkly_feature_flag" "new_navigation" {
  project_key = var.project_key
  key         = "new-navigation-bar"
  name        = "New Navigation Bar"
  description = "Redesigned navigation with improved UX"
  
  variation_type = "boolean"
  
  client_side_availability {
    using_environment_id = true
    using_mobile_key     = false
  }
  
  # This flag belongs to the frontend team view
  view_keys = [var.team_view_key]
  
  tags = ["frontend", "navigation", "ux"]
}

resource "launchdarkly_feature_flag" "maintenance_banner" {
  project_key = var.project_key
  key         = "maintenance-banner"
  name        = "Maintenance Banner"
  description = "Display maintenance notification banner"
  
  variation_type = "string"
  
  variations {
    value = ""
    name  = "No banner"
  }
  variations {
    value = "Scheduled maintenance tonight at 2 AM EST"
    name  = "Scheduled"
  }
  variations {
    value = "Emergency maintenance in progress"
    name  = "Emergency"
  }
  
  defaults {
    on_variation  = 1
    off_variation = 0
  }
  
  client_side_availability {
    using_environment_id = true
    using_mobile_key     = true
  }
  
  # This banner is relevant to frontend team but also visible as a shared feature
  view_keys = [
    var.team_view_key,
    var.shared_view_key
  ]
  
  tags = ["frontend", "maintenance", "banner"]
}

