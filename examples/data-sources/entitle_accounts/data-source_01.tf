# With additional filter
data "entitle_accounts" "search" {
  integration_id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
  filter {
    search = "example@beyondtrust.com"
  }
}

output "accounts" {
  value = data.entitle_accounts.search.accounts
}
