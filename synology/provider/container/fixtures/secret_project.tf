terraform {
  required_providers {
    synology = {
      source = "florianehmke/synology"
    }
  }
}

resource "synology_container_project" "default" {
  name = "secrets project"

  secrets = {
    "foo" = {
      name    = "foo"
      content = "bar"
      file    = "foo.txt"
    }
  }

  run = false
}
