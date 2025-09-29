# No filters: get all directory_groups for integration
data "entitle_directory_groups" "all" {
}

# With additional filter
data "entitle_directory_groups" "search" {
  filter {
    search = "example"
  }
}

output "directory_groups" {
  value = data.entitle_directory_groups.search.directory_groups
}
