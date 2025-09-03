# No filters: get all users
data "entitle_users" "all" {}

# With filter block
data "entitle_users" "search" {
  filter {
    search = "example@beyondtrust.com"
  }
}

output "users" {
  value = data.entitle_users.search.users
}
