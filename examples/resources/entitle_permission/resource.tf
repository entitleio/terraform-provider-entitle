data "entitle_permissions" "example_get_role_permissions" {
  filter {
    role_id = "d080bfa-9143-11ee-b9d1-0242ac120002"
  }
}

resource "entitle_permission" "example_role_permissions" {
  for_each = {
    for p in data.entitle_permissions.example_get_role_permissions.permissions :
    p.id => p
  }

  id = each.value.id
}