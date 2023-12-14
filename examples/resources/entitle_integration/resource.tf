resource "entitle_integration" "example" {
  connection_json = "{\"token\": \"PUT_YOUR_SLACK_TOKEN\",\"options\": {\"plan\": \"PUT_YOUR_SLACK_PLAN\"}}"
  name            = "example"
  application = {
    name = "Slack"
  }
  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }
  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
  }
  allowed_durations       = [3600]
  maintainers             = []
  allow_creating_accounts = false
}