terraform {
  required_providers {
    godaddy = {
      version = "2.0.0"
      source  = "github.com/andrewstucki/godaddy"
    }
  }
}

provider "godaddy" {
  baseurl = "https://api.ote-godaddy.com"
}

resource "godaddy_domain_purchase" "gd-check" {
  registrant {
    address {
      line_1      = "1234 hohoho way"
      city        = "alameda"
      state       = "California"
      country     = "US"
      postal_code = "94502"
    }
    email      = "domains@example.com"
    first_name = "test"
    last_name  = "test"
    phone      = "+1.1111111111"
  }
  tech {
    address {
      line_1      = "1234 hohoho way"
      city        = "alameda"
      state       = "California"
      country     = "US"
      postal_code = "94502"
    }
    email      = "domains@example.com"
    first_name = "test"
    last_name  = "test"
    phone      = "+1.1111111111"
  }
  billing {
    address {
      line_1      = "1234 hohoho way"
      city        = "alameda"
      state       = "California"
      country     = "US"
      postal_code = "94502"
    }
    email      = "domains@example.com"
    first_name = "test"
    last_name  = "test"
    phone      = "+1.1111111111"
  }
  admin {
    address {
      line_1      = "1234 hohoho way"
      city        = "alameda"
      state       = "California"
      country     = "US"
      postal_code = "94502"
    }
    email      = "domains@example.com"
    first_name = "test"
    last_name  = "test"
    phone      = "+1.1111111111"
  }
  enable_privacy = false
  auto_renew     = true
  years_leased   = 1
  domain         = "1456725.com"

  lifecycle {
    ignore_changes = all
  }
}

