resource "keboola_branch" "branch_test" {
  name = "branch-test"
}

resource "keboola_branch_metadata" "description" {
  branch_id = keboola_branch.branch_test.id
  key = "description"
  value = "This is a test branch"
}
