resource "entitle_bundle" "example" {
  name        = "example"
  category    = "terraform"
  description = "terraform"
  tags        = ["terraform"]
  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }
  roles = [{
    id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
  }]
  allowed_durations = [3600]
}