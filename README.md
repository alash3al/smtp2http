SMTP2HTTP (email-to-web)
========================
This repo is a fork from the v3.0.1 of @alash3al/smtp2http. 
It's been initially created to fix accent characters encoding issue when receiving emails from Microsoft french outlook mailbox, which use a specific charset (Windows-1252).

smtp2http is a simple smtp server that resends the incoming email to the configured web endpoint (webhook) as a basic http post request.

Depedencies : 
* github.com/go-resty/resty/v2 v2.3.0
* github.com/miekg/dns v1.1.50
* github.com/stouch/go-smtpsrv
* golang.org/x/crypto

Dev 
===
- `go mod vendor`
- `go build`

Dev with Docker
==============
- `go mod vendor`
- `docker build -f Dockerfile.dev -t smtp2http-dev .`
- `docker run -p 25:25 --timeout.read=50 --timeout.write=50 --webhook=http://some.hook/api smtp2http-dev`

or build it as it comes from stouch repo :
- `go mod vendor`
- `docker build -t smtp2http .`
- `docker run -p 25:25 --env --timeout.read=50 --timeout.write=50 --webhook=http://some.hook/api smtp2http`

timeout options are of course optional but make it easier to test in local with `telnet localhost 25`
Here is a telnet example payload : 
```
HELO zeus
# smtp answer

MAIL FROM:<email@from.com>
# smtp answer

RCPT TO:<youremail@example.com>
# smtp answer

DATA
your mail content
.

```

Docker (production)
=====
**Docker images arn't available online for now**
**See "Dev with Docker" above**
- `docker run -p 25:25 smtp2http --webhook=http://some.hook/api`

Native usage
=====
`smtp2http --listen=:25 --webhook=http://localhost:8080/api/smtp-hook`
`smtp2http --help`

Contribution
============
Original repo from @alash3al
Thanks to @aranajuan


