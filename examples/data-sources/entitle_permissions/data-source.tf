# No filters: get all permissions
data "entitle_permissions" "all" {}

# With filter block
data "entitle_permissions" "account_permissions" {
  filter {
    account_id = "d080bfa-9143-11ee-b9d1-0242ac120002"
  }
}

data "entitle_permissions" "integration_permissions" {
  filter {
    integration_id = "d080bfa-9143-11ee-b9d1-0242ac120002"
  }
}

data "entitle_permissions" "role_permissions" {
  filter {
    role_id = "d080bfa-9143-11ee-b9d1-0242ac120003"
  }
}