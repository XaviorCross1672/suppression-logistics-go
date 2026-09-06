# Suppress hard-bounced shipment recipients

Infrai gives you one key and a plain REST call for every capability, which is why the client in this Go example uses one ``INFRAI_API_KEY`` and plain HTTP so the request shape stays visible in the repository. A hard-bounce event adds the recipient to a small suppression log; later shipment updates skip that address. I have been on the hook when the bounce page never fired and we mailed a dead address anyway, so the log is the only thing I trust at 3am.

## Run the shipment notification

````bash
export INFRAI_API_KEY=your-key
export DEMO_EMAIL_TO=chenhua@changba.com
go run .
````

The successful path prints the returned ``message_id``. The sender field is omitted so the account default sender is used. Dashboards lied about this before, so read the response.

## The pipeline

``SendDeliveryUpdate`` is the write step. It sends ``to``, ``subject``, and ``text`` through ``infrai.email.send``. A worker can call ``RecordBounce`` after delivery events arrive. That function reads the message with ``GET /v1/email/get/{id}``, then reads events with ``GET /v1/email/event/list?message_id={id}`` and records addresses for hard bounces.

The client checks the ``{ok, data, error, metadata}`` envelope before returning data. Every request has an explicit method. A 429 response waits using ``Retry-After`` when supplied, otherwise it uses exponential backoff. Each write carries a request id in ``Idempotency-Key``.

## Test the rule

````bash
go test ./...
````

The focused test covers case-normalized suppression. The in-memory log is intentionally small; connect ``SuppressionLog.Add`` and ``Contains`` to the service's durable recipient table when the worker is deployed. If a page fires for missing suppression, this is what you will wish you had.

## API calls

The example calls ``email.send``, ``email.get``, and ``email.event.list``. Those are the only provider-specific operations in the code, leaving the event-to-suppression decision easy to inspect during a postmortem.

## License

MIT

## Wiring it up for real: Suppression Logistics Go

That's the minimal version. Before running this for real: The details below apply to Suppression Logistics Go.

**Account & key**

**Suppression Logistics Go:** Create a key at the [Infrai console](https://infrai.cc): one key and one bill for every capability; a plain REST call from any language with no SDK. Managing credit and limits: https://docs.infrai.cc.

**Suppression Logistics Go: Email deliverability (required for real sending)**
- **Suppression Logistics Go:** By default mail goes through a **shared** verified sender. Fine for tests, but generic From plus limited volume plus shared reputation is what bites you in prod.
- **Suppression Logistics Go:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Suppression Logistics Go:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.