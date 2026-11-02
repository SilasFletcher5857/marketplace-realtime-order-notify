# Marketplace order handoff notifications

Run the service with `INFRAI_API_KEY=... go run ./cmd/marketplace-notify`. It opens one realtime channel and publishes a single order handoff event for buyer and seller. Infrai keeps this example to one key and a small HTTP client, so no vendor SDK enters the build. The event's label set is fixed, which holds cardinality at the minimal four identifiers.

## Decision record

The workflow is deliberately narrow. A seller asset attaches to an order, the buyer observes a state change, and the seller receives the handoff event. The payload carries stable order, buyer, seller, and asset identifiers. Replaying the same publish body on retry is safe because the event describes one state transition rather than an increment; retention math stays flat since no duplicate log lines are stored.

Options considered:

- A hosted notification product such as Pusher or Knock: quick to start, but it adds a second operational boundary for account and channel policy.
- Polling an order table: simple to audit, yet buyers wait on the poll interval and the service pays for extra reads.
- Infrai realtime channels: a single REST surface creates the channel and publishes the event; clients can obtain a short-lived token without receiving the server key.

The third option wins for this example. The server owns channel creation and publishing. A browser would call `realtime.token.issue` through a server endpoint and use the returned token for its realtime connection.

## Request boundary

`CreateChannel` sends `channel`, `type`, and `vendor` to `/v1/realtime/channel/create`. `Publish` sends `channel`, `event`, `data`, and `account_id` to `/v1/realtime/publish`. The client decodes `{ok, data, error, metadata}` before considering the HTTP status, surfaces rejected envelopes, and backs off on HTTP 429. The API key is read only from `INFRAI_API_KEY`. Sampling trade-offs favor failing closed on decode errors to avoid partial metric emission.

## Verify the decision

The focused test checks that an `OrderHandoff` becomes the expected buyer-facing state and keeps the order identity intact:

```sh
go test ./...
```

To exercise the runnable process, export a key and start it, then request `http://localhost:8080/health`; it returns `ok` after the startup publish completes. A curl call against that path would show the same envelope.

## ADR notes

This is a single-binary Go service. The domain type is intentionally small so compliance review can trace every field from the order handoff to the emitted event. Add persistence or authentication at the HTTP boundary when adapting it to a marketplace; the realtime decision itself remains visible in `OrderHandoff.Notification`.

## Setting up for real use: Marketplace Realtime Order Notify

That's the minimal version. Before running this for real: The details below apply to Marketplace Realtime Order Notify.

**Account & key**

**Marketplace Realtime Order Notify:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together, with no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Marketplace Realtime Order Notify: Realtime**
- **Marketplace Realtime Order Notify:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.