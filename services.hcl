services {
  name = "foo"
  port = 1234
  connect {
    sidecar_service{
      proxy {
        mode = "transparent"
      }
    }
  }
}

services {
  name = "bar"
  address = "10.10.10.3"
  port = 2345
  connect {
    sidecar_service{
      proxy {
        mode = "transparent"
      }
    }
  }
}

services {
  name = "baz"
  address = "10.10.2.3"
  port = 3456
  connect {
    sidecar_service{
      proxy {
        mode = "transparent"
      }
    }
  }
}
services {
  name = "qux"
  address = "10.1.2.3"
  port = 9876
  connect {
    sidecar_service{
      proxy {
        mode = "transparent"
      }
    }
  }
}