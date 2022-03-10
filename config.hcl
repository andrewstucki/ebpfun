// L7 rules service 1
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

// L7 rules service 2
ingress {
  address = "127.0.0.1"
  port = 8001
  http = true
}

exemption {
  source = "127.0.0.1"
  destination = "127.0.0.1"
  port = 8001

  l7_rule {
    header_present = "x-bar"
  }
}

// No L7 offload
ingress {
  address = "127.0.0.1"
  port = 8002
}

exemption {
  source = "127.0.0.1"
  destination = "127.0.0.1"
  port = 8002
}