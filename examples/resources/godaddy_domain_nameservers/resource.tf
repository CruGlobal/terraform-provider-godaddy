resource "godaddy_domain_nameservers" "example" {
  domain      = "example.com"
  nameservers = ["ns7.example.com", "ns8.example.com"]
}
