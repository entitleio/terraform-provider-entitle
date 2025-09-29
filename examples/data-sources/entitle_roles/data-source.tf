# No filters: get all roles for resource
data "entitle_roles" "all" {
  resource_id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
}

# With additional filter
data "entitle_roles" "search" {
  resource_id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
  filter {
    search = "example@beyondtrust.com"
  }
}

output "users" {
  value = data.entitle_roles.search.roles
}
