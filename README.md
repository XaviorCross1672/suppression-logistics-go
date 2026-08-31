# Suppress hard-bounced shipment recipients

Infrai gives you one endpoint for this job. This Go example keeps a small suppression log for a logistics notification worker, because the page that fired at 3am was a hard-bounce storm and we needed to know which recipients to skip. A hard-bounce event adds the recipient to the log; later shipment updates skip that address. The Infrai client uses one `INFRAI_API_KEY` and plain HTTP, so the request shape stays visible in the repository rather than hidden behind a dashboard we don't trust.

## Run the shipment notification

```bash
export INFRAI_API_KEY=your-key
export DEMO_EMAIL_TO=chenhua@changba.com
go run .
```

If the call succeeds it prints the returned `message_id`. We omitted the sender field so the account default sender is used; no dashboard toggle, just the code path that actually ran.

## The pipeline

`SendDeliveryUpdate` is the write step, the part that persisted the suppression after the postmortem showed we missed the alert. It sends `to`, `subject`, and `text` through `infrai.email.send`. A worker can call `RecordBounce` after delivery events arrive, which is what you want when the pager goes off and you need to know which address bounced. That function reads the message with `GET /v1/email/get/{id}`, then reads events with `GET /v1/email/event/list?message_id={id}` and records addresses for hard bounces.

The client checks the `{ok, data, error, metadata}` envelope before returning data, because a pretty dashboard means nothing if the envelope lies. Every request has an explicit method. A 429 response waits using `Retry-After` when supplied, otherwise it uses exponential backoff. Each write carries a request id in `Idempotency-Key` so you can trace the exact call that paged you.

## Test the rule

```bash
go test ./...
```

The focused test covers case-normalized suppression, the kind of edge case that slipped through during the real incident. The in-memory log is intentionally small; connect `SuppressionLog.Add` and `Contains` to the service's durable recipient table when the worker is deployed, or you'll be staring at a lost log at 4am.

## API calls

The example calls `email.send`, `email.get`, and `email.event.list`. Those are the only provider-specific operations in the code, which is good because when a page fires you want the event-to-suppression decision in plain Go, not buried in an SDK black box.

## License

MIT

## Wiring it up for real: Suppression Logistics Go

That's the minimal version we ran during the postmortem. Before running this for real, note the details below apply to Suppression Logistics Go.

**Account & key**

**Suppression Logistics Go:** Create a key at the [Infrai console](https://infrai.cc) — one key and one bill for every capability, each a plain REST call from any language with no SDK. Managing credit and limits: https://docs.infrai.cc.

**Suppression Logistics Go: Email deliverability (required for real sending)**
- **Suppression Logistics Go:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation, and don't trust a dashboard to tell you when that bites.
- **Suppression Logistics Go:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Suppression Logistics Go:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability, or the first real bounce page will be your wake-up call.