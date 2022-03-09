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

// treat as a TCP server
ingress {
  address = "127.0.0.1"
  port = 8002
}

exemption {
  source = "127.0.0.1"
  destination = "127.0.0.1"
  port = 8002
}

// No exemptions
ingress {
  address = "127.0.0.1"
  port = 8003
}

// TCP server
ingress {
  address = "10.0.2.15"
  port = 8004
}

ingress {
  address = "127.0.0.1"
  port = 8004
}

exemption {
  source = "127.0.0.1"
  destination = "127.0.0.1"
  port = 8004
}

// UDP echo
ingress {
  address = "10.0.2.15"
  port = 8005
}

ingress {
  address = "127.0.0.1"
  port = 8005
}

exemption {
  source = "127.0.0.1"
  destination = "10.0.2.15"
  port = 8005
}