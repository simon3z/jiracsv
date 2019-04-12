# Jira to CSV Export Tool

## Building

    $ go build

## Examples

In order to avoid your password being stored in the bash history it is
strongly suggested to export an environment variable:

    $ read -p Password: -s PASSWORD && echo && export PASSWORD

Collecting the issues for multiple components in the same project and
version:

    $ ./jiracsv -h <jiraurl> -u <username> -p <project> -v <version> -c <component1> -c <component2> ...
