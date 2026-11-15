# SMTP2HTTP (email-to-web)

`smtp2http` is a simple SMTP server that resends each incoming email to a
configured web endpoint (webhook) as an HTTP POST request with a JSON body.

## Usage

```
smtp2http --listen=:25 --webhook=http://localhost:8080/api/smtp-hook
smtp2http --help
```

| Flag | Default | Description |
|---|---|---|
| `--listen` | `:smtp` (port 25) | address to listen on |
| `--webhook` | `http://localhost:8080/my/webhook` | where to POST each message |
| `--domain` | *(empty: accept any)* | only accept mail addressed to this domain |
| `--name` | `smtp2http` | SMTP banner hostname |
| `--msglimit` | `2097152` (2 MiB) | maximum incoming message size |
| `--timeout.read` | `5` | read timeout in seconds |
| `--timeout.write` | `5` | write timeout in seconds |
| `--log.level` | `info` | `debug`, `info`, `warn` or `error` |
| `--log.format` | `text` | `text` or `json` |
| `--user`, `--pass` | — | **deprecated, ignored.** See below. |

### Restricting recipients

`--domain` is off by default, which accepts mail for *any* recipient domain.
Set it to reject everything else:

```
smtp2http --listen=:25 --domain=example.com --webhook=http://localhost:8080/api/smtp-hook
```

The comparison is case-insensitive. Subdomains do not match — `--domain=example.com`
rejects `user@mail.example.com`.

### A note on authentication

This server does **not** authenticate senders. `--user` and `--pass` have never
done anything; they are still accepted so that older deployments keep starting,
but they log a warning and are otherwise ignored. SPF is evaluated and reported
in the payload, but DKIM and DMARC are not checked.

Run it behind a firewall, or for appliances on a LAN that only speak SMTP. Think
carefully before pointing an internet-facing MX at it.

### Logging

Structured, on stderr. Every message is logged once, as `accepted` or
`rejected`; rejections carry a `reason` so the three causes are distinguishable:

```
level=INFO msg=accepted from=a@sender.tld to=bob@example.com message_id=... spf=none attachments=1 duration_ms=3
level=WARN msg=rejected ... reason=to_domain_not_allowed
level=ERROR msg=rejected ... reason=webhook_unreachable err="..."
level=ERROR msg=rejected ... reason=webhook_status status=500
```

Subjects and bodies are never logged; `subject_len` is recorded instead. The
webhook URL is reduced to `scheme://host` wherever it is logged -- for most
providers the path or query *is* the credential.
`--log.format=json` emits one JSON object per line.

How a rejection is answered depends on whose fault it is:

| Cause | SMTP reply | Sender's behaviour |
|---|---|---|
| webhook unreachable, or answered anything but `200` | `451 4.3.0` | keeps the mail queued and retries |
| recipient outside `--domain` | `550 5.7.1` | bounces |
| message cannot be parsed | `554 5.6.0` | bounces |

So a webhook outage or a deploy of the receiving application delays mail
instead of losing it. Up to 1.1.1 all three were a permanent `554`.

## Docker

```
docker run -p 25:25 oskarspakers/smtp2http --webhook=http://some.hook/api
```

Images are published for `linux/amd64` and `linux/arm64`.

| Tag | Tracks |
|---|---|
| `latest`, `X.Y.Z`, `X.Y` | the newest release |
| `edge`, `sha-<short>` | the newest commit on `main` (unstable) |

Port 25 is only required on the *host* side — MX records carry no port number, so
you can bind an unprivileged port inside the container:

```
docker run -p 25:2525 oskarspakers/smtp2http --listen=:2525 --webhook=http://some.hook/api
```

## Development

```
go build
go test ./...
```

With Docker:

```
docker build -f Dockerfile.dev -t smtp2http-dev .
docker run -p 25:25 smtp2http-dev --timeout.read=50 --timeout.write=50 --webhook=http://some.hook/api
```

Or the production image:

```
docker build -t smtp2http .
docker run -p 25:25 smtp2http --timeout.read=50 --timeout.write=50 --webhook=http://some.hook/api
```

The `timeout` options are optional, but generous values make it easier to poke at
the server by hand with `telnet localhost 25`:

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

## Contribution

Original repo from [@alash3al](https://github.com/alash3al/smtp2http).
Thanks to [@aranajuan](https://github.com/aranajuan).
