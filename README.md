# Jira to CSV Export Tool

## Building

    $ go build

## Examples

In order to avoid your password being stored in the bash history it is strongly suggested to export an environment variable:

    $ read -p Password: -s PASSWORD && echo && export PASSWORD

Collecting the issues for multiple components in the same project and version:

    $ ./jiracsv -u <username> -c <config-file> -p <profile-id>

Configuration file example:

    instance:
      url: https://jira.atlassian.com
    profiles:
    - id: jira-latest-fixes
      jql:
        project = JRASERVER AND
        fixVersion = latestReleasedVersion()
      components:
        exclude:
        - Tomcat
