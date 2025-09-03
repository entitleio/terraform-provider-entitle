# No filters: get all resources for integration
data "entitle_resources" "all" {
  integration_id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
}

# With additional filter
data "entitle_resources" "search" {
  integration_id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
  filter {
    search = "example"
  }
}

output "resources" {
  value = data.entitle_resources.search.resources
}
