terraform {
  required_providers {
    pfsense = {
      source = "registry.terraform.io/elacy/pfsense"
    }
  }
}

variable "pfsense_url" {
  type    = string
  default = "https://192.168.1.1"
}

variable "pfsense_password" {
  type      = string
  sensitive = true
}

provider "pfsense" {
  url      = var.pfsense_url
  username = "admin"
  password = var.pfsense_password

  # Uncomment for self-signed certificates:
  # skip_tls_verify = true
}

resource "pfsense_firewall_alias" "webservers" {
  name    = "webservers"
  type    = "host"
  descr   = "Production web servers"
  address = ["10.0.0.10", "10.0.0.11", "10.0.0.12"]
}

resource "pfsense_firewall_rule" "allow_https" {
  descr            = "allow-https-to-webservers"
  type             = "pass"
  interface        = ["lan"]
  ipprotocol       = "inet"
  protocol         = "tcp"
  source           = "any"
  destination      = "webservers"
  destination_port = "443"
}

resource "pfsense_routing_gateway" "wan_gw" {
  name      = "WAN_GW"
  interface = "wan"
  gateway   = "203.0.113.1"
}

resource "pfsense_routing_static_route" "rfc1918" {
  network = "10.99.0.0/16"
  gateway = "WAN_GW"
  descr   = "Route to internal lab network"
}

resource "pfsense_system_tunable" "ip_forwarding" {
  tunable = "net.inet.ip.forwarding"
  value   = "1"
  descr   = "Enable IP forwarding"
}

data "pfsense_firewall_aliases" "all" {}

output "alias_names" {
  value = [for a in data.pfsense_firewall_aliases.all.aliases : a.name]
}
