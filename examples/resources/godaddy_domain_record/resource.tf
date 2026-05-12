resource "godaddy_domain_record" "example" {
  domain = "example.com"

  record = [
    {
      name = "www"
      type = "CNAME"
      data = "example.github.io"
      ttl  = 3600
    },
    {
      name     = "@"
      type     = "MX"
      data     = "aspmx.l.google.com."
      ttl      = 600
      priority = 1
    },
  ]

  addresses   = ["192.168.1.2", "192.168.1.3"]
  nameservers = ["ns7.example.com", "ns8.example.com"]
}
