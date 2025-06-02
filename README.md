# baton-expensify

`baton-expensify` is a connector for Expensify built using the [Baton SDK](https://github.com/conductorone/baton-sdk). It communicates with the Expensify API to sync data about policies and users.

Check out [Baton](https://github.com/conductorone/baton) to learn more the project in general.

# Getting Started

## Prerequisites

1. Pair of credentials to use the API. You can find `partnerUserID` and `partnerUserSecret` [here](https://www.expensify.com/tools/integrations)

## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-expensify
baton-expensify
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_PARTNER_USER_ID=partnerUserId BATON_PARTNER_USER_SECRET=partnerUserSecret ghcr.io/conductorone/baton-expensify:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source

```
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-expensify/cmd/baton-expensify@main

BATON_PARTNER_USER_ID=partnerUserId BATON_PARTNER_USER_SECRET=partnerUserSecret
baton resources
```

# Data Model

`baton-expensify` will pull down information about the following Expensify resources:

- Policies
- Users

# Contributing, Support and Issues

We started Baton because we were tired of taking screenshots and manually building spreadsheets. We welcome contributions, and ideas, no matter how small -- our goal is to make identity and permissions sprawl less painful for everyone. If you have questions, problems, or ideas: Please open a Github Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-expensify` Command Line Usage

```
baton-expensify

Usage:
  baton-expensify [flags]
  baton-expensify [command]

Available Commands:
  capabilities       Get connector capabilities
  completion         Generate the autocompletion script for the specified shell
  help               Help about any command

Flags:
      --client-id string             The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string         The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
  -f, --file string                  The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                         help for baton-expensify
      --log-format string            The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string             The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
      --partner-user-id string       The Expensify partner user id used to connect to the Expensify API. ($BATON_PARTNER_USER_ID)
      --partner-user-secret string   The Expensify partner user secret used to connect to the Expensify API. ($BATON_PARTNER_USER_SECRET)
  -p, --provisioning                 This must be set in order for provisioning actions to be enabled. ($BATON_PROVISIONING)
  -v, --version                      version for baton-expensify

Use "baton-expensify [command] --help" for more information about a command.
```

