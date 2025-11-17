resource "entitle_workflow" "example" {
  name = "example"
  rules = [
    {
      any_schedule = true
      approval_flow = {
        steps = [
          {
            approval_entities = [
              {
                type = "Automatic"
              }
            ]
            operator   = "or"
            sort_order = 1
          }
        ]
      }

      sort_order     = 1
      under_duration = 3600
    }
  ]
}