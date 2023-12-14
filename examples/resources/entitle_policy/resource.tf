resource "entitle_policy" "example" {
  in_groups = [{
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }]

  roles = [{
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }]
}