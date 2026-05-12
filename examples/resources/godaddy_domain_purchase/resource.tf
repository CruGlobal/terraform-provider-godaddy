resource "godaddy_domain_purchase" "example" {
  domain       = "example.com"
  years_leased = 1

  enable_privacy = false
  auto_renew     = true

  registrant = {
    address = {
      line_1      = "1234 Main St"
      city        = "Alameda"
      state       = "California"
      country     = "US"
      postal_code = "94502"
    }
    email      = "domains@example.com"
    first_name = "Jane"
    last_name  = "Doe"
    phone      = "+1.1111111111"
  }

  # admin / tech / billing blocks accept the same shape.

  lifecycle {
    # Many GoDaddy fields cannot be changed once a domain is registered;
    # ignore drift to avoid spurious diffs.
    ignore_changes = all
  }
}
