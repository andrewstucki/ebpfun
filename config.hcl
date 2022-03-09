ingress {
  address = "127.0.0.1"
  port = 8000
  http = true
}

exemption {
  source = "127.0.0.1"
  destination = "127.0.0.1"
  port = 8000

  l7_rule {
    header_present = "x-foo"
  }
}