# 🐝 wasp

AWS SSO Profiles CLI

Built with [Bubble Tea v2](https://charm.land/blog/v2/), [Lip Gloss v2](https://charm.land/lipgloss/), and [Bubbles v2](https://charm.land/bubbles/) from the [Charm](https://charm.land) ecosystem.

## Requirements

- Go 1.24.2 or later

## Installation

Have go installed, then run

```
go install github.com/buzzsurfr/wasp@latest
```

## Getting started

You need to have a SSO session in your AWS config file. If you don't have one, you can create one using the following command:

```
aws configure sso-session
```

Then run this command to get the starter interface:

```
wasp init
```

## Profile Switching

You can switch profiles (with the context around SSO sessions) by running

```
eval $(wasp switch)
```
