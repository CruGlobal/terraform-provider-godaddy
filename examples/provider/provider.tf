terraform {
  required_providers {
    godaddy = {
      source  = "CruGlobal/godaddy"
      version = "~> 2.0"
    }
  }
}

provider "godaddy" {
  # key and secret may also be supplied via GODADDY_API_KEY / GODADDY_API_SECRET.
  key    = "your-api-key"
  secret = "your-api-secret"
}
