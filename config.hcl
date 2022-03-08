ingress {
  address = "127.0.0.1"
  port = 80
}

ingress {
  address = "127.0.0.1"
  port = 81
}

ingress {
  address = "127.0.0.1"
  port = 82
}

exemption {
  source = "127.0.0.1"
  destination = "127.0.0.1"
  port = 80
}

exemption {
  source = "127.0.0.1"
  destination = "127.0.0.1"
  port = 82
}
