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
                value = {
                  approval = null
                }
              }
            ]
            notified_entities = []
            operator          = "or"
            sort_order        = 1
          }
        ]
      }

      in_groups      = []
      in_schedules   = []
      sort_order     = 1
      under_duration = 3600
    }
  ]
}