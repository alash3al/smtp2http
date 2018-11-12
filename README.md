SMTP2HTTP (email-to-web)
========================
smtp2http is a simple smtp server that resends the incoming email to the configured web endpoint (webhook) as a basic http post request.

Why
===
At our company `uFlare` we wanted to build a platform form receiving requests via mai clients and posting it to a customized webhook to do its business logic, we wanted also to use `Go` as the environment so we started to develop that smtp server based on 
[go-smtpsrv](https://github.com/alash3al/go-smtpsrv) library, and because we believe in the power of the opensource we decided to release this software as an opensource project for the community because it may help anyone else.

Installation
=============
- binaries: go to [releases](https://github.com/uflare/smtp2http/releases) page and choose your distribution.
- go: `go get github.com/uflare/smtp2http`

Usage
=====
`smtp2http --listen=:25 --webhook=http://localhost:8080/api/smtp-hook --strict=true`

Help
====
`smtp2http --help`

Contribution
=============
> Fork > Patch > Create Pull Request

Author
=======
[uFlare](https://www.uflare.io)