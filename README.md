SMTP2HTTP (email-to-web)
========================
smtp2http is a simple smtp server that resends the incoming email to the configured web endpoint (webhook) as a basic http post request.

Installation
=============
- docker: `docker run -p 25:25 smtp2http --webhook=http://some.hook/api`
- binaries: go to [releases](https://github.com/alash3al/smtp2http/releases) page and choose your distribution.
- go: `go get github.com/alash3al/smtp2http`

Usage
=====
`smtp2http --listen=:25 --webhook=http://localhost:8080/api/smtp-hook`

Help
====
`smtp2http --help`

Contribution
=============
> Fork > Patch > Create Pull Request

