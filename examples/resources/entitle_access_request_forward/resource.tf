resource "entitle_access_request_forward" "example" {
  forwarder = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }
  target = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
  }
}