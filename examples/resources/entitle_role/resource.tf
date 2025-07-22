resource "entitle_role" "example" {
  name = "My Role Example"
  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
  }
  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120004"
  }
  requestable = true
  allowed_durations = [-1]
}