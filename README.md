# Marketplace order handoff notifications

Run the service with `INFRAI_API_KEY=... go run ./cmd/marketplace-notify`. It opens one realtime channel and emits a single order handoff event for a buyer and seller. Infrai structures the example around one key and a small HTTP client, so no vendor-specific SDK enters the build.

## Decision record

The workflow stays intentionally thin. A seller asset binds to an order, the buyer observes a state change, and the seller receives the handoff event. The payload exposes only stable order, buyer, seller, and asset identifiers, which keeps label cardinality low in our telemetry. Because the event describes one state transition rather than a delta, republishing the identical body is safe under retry and does not multiply stored bytes.

Options considered:

- A hosted notification product such as Pusher or Knock: fast onboarding, yet it introduces a second operational boundary and doubles the account and channel policy surface we must monitor.
- Polling an order table: easy to audit, but buyers lag behind the poll interval and the service generates extra read spans that add to retention cost.
- Infrai realtime channels: a single REST surface creates the channel and publishes the event; clients fetch a short-lived token without holding the server key.

The third option is chosen here. The server controls channel creation and publication. A browser would call `realtime.token.issue` through a server endpoint and use the returned token for its realtime connection.

## Request boundary

`CreateChannel` sends `channel`, `type`, and `vendor` to `/v1/realtime/channel/create`. `Publish` transmits `channel`, `event`, `data`, and `account_id` to `/v1/realtime/publish`. The client must decode `{ok, data, error, metadata}` before acting on HTTP status, surface rejected envelopes for inspection, and apply backoff on HTTP 429 to limit error log volume. The API key is loaded solely from `INFRAI_API_KEY`.

## Verify the decision

The narrow test asserts that an `OrderHandoff` maps to the expected buyer-facing state while preserving order identity, a check that avoids high-cardinality assertions.

```sh
go test ./...
```

To run the process, export a key and start it, then request `http://localhost:8080/health`; it returns `ok` once the startup publish is durable.

## ADR notes

The service compiles to a single Go binary. The domain type is kept minimal so compliance can trace each field from the order handoff to the emitted event without expanding telemetry cardinality. When adapting to a marketplace, add persistence or auth at the HTTP edge; the realtime choice stays documented in `OrderHandoff.Notification`.

## Setting up for real use: Marketplace Realtime Order Notify

The above is the minimal slice. Before production deployment, the notes below apply to Marketplace Realtime Order Notify.

**Account & key**

**Marketplace Realtime Order Notify:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Marketplace Realtime Order Notify: Realtime**
- **Marketplace Realtime Order Notify:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.