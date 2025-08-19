resource "entitle_resource" "my_resource" {
  name                     = "example name"
  user_defined_description = "example user defined description"
  requestable              = true
  allowed_durations        = [-1]
  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }
  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
  }
  integration = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120004"
  }
  maintainers = [
    {
      type = "user"
      entity = {
        id = "7d080bfa-9143-11ee-b9d1-0242ac120005"
      }
    }
  ]
  user_defined_tags = [
    "example1",
    "example2"
  ]
  prerequisite_permissions = [
    {
      default = true
      role = {
        id = "7d080bfa-9143-11ee-b9d1-0242ac120006"
      }
    }
  ]
}