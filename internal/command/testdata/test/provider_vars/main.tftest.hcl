
variables {
  resource_directory = "my-resource-dir"
}

provider "test" {
  resource_directory = var.resource_directory
}

run "test" {}
