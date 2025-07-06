# Things to do

* Have the AWS SSO flow use the same OIDC method that the AWS CLI v2 moved to.
* Improvements for `ssm` command:
  * Add JSON output for the `get` and `list` commands.
  * Handle getting and putting tags.
  * Implement `copy` command.
* Improve / add tests for the AWS code once I work out how to mock AWS services.
* Add --version flags to everything? Since they are all in the 1 repo have it use CalVer (YYYY.0M.0D) at build time perhaps?
