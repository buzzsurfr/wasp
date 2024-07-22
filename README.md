# 🐝 wasp

AWS SSO Profiles CLI

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