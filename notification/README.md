# Notification

Send a business notification through multiple channels without coupling the
business operation to a mail provider, ORM, or queue backend.

The module includes synchronous dispatch, a typed mail channel, a JSON inbox
channel, and a process-local memory inbox. It does not include SMTP, GORM,
durable queuing, automatic retries, or a global default manager.

## Installation

```sh
go get github.com/go-fries/fries/notification/v4
```

All channel packages shown here belong to this same module.

## Recommended usage

Construct channels and a `notification.Manager` at application startup and
inject the manager into business services. Given an initialized `notifier`,
recipient `to`, and an application notification implementing the contracts below:

```go
result, err := notifier.Send(ctx, to, InvoicePaid{InvoiceID: "INV-1001"})
```

`Send` waits for every attempted channel. Always inspect the error and, on
partial failure, `result.Deliveries` before deciding what to retry. A successful
mail delivery must not be repeated merely because another channel failed.

See the [complete, compile-checked example](example_test.go) for an invoice
notification delivered through mail and an inbox, including setup, error handling,
and read state. Run it from the repository root:

```sh
make test/notification ARGS='-run=Example -v'
```

The example sender prints instead of contacting a mail service. The application
owns and closes real sender/storage clients; Manager starts no background work
and has nothing to stop or close.

## Define a notification and recipient

A notification implements:

```go
type Notification interface {
	Type() string
	Via(ctx context.Context, to Recipient) ([]string, error)
}
```

Use an explicit business type, such as `invoice.paid.v1`. `Via` returns registered
channel names, for example `[]string{mail.Name, database.Name}`. It can inspect
recipient preferences and return lookup errors. An empty list is a successful
no-op; duplicate names are sent once in first-seen order.

Each recipient implements `NotificationKey() notification.RecipientKey`.
The key contains `TenantID`, entity `Type`, and business `ID`. Type and ID are
required for an inbox; TenantID may be empty in single-tenant applications.
Keys are case-sensitive UTF-8 text. A lightweight application adapter can hold
these fields without adding notification dependencies to a database model.
Anonymous recipients return a zero key; partially populated keys are rejected
before channel selection or delivery.

Add only the capabilities needed by the selected channels:

| Package | Notification method | Recipient method |
| --- | --- | --- |
| `mail` | `ToMail(context.Context, notification.Recipient) (mail.Message, error)` | `MailRoute(context.Context) (net/mail.Address, error)` |
| `database` | `ToDatabase(context.Context, notification.Recipient) (any, error)` | A valid `NotificationKey()` |

`ToDatabase` may return a typed application struct with JSON tags. The channel
encodes it and stores a display snapshot. `ToMail` returns a subject and plain
text and/or HTML bodies; the injected sender owns MIME construction, From,
credentials, and transport defaults. Neither channel provides a template engine.

Missing message capabilities or routes are errors, not silent skips. An optional
`notification.SendGuard` provides
`ShouldSend(ctx, recipient, channel) (bool, error)`: false with no error explicitly
skips that channel, and an error fails that channel. It runs before rendering.

Notifications and recipients must not be mutated while a Send is in progress.
Shared channels, senders and stores must support concurrent calls.

## Configure channels

Given an application-owned `mailer` implementing `mail.Sender`, and a `store`
implementing `database.Writer`:

```go
mailChannel, err := mail.NewChannel(mailer)
if err != nil {
	return err
}
databaseChannel, err := database.NewChannel(store)
if err != nil {
	return err
}
notifier, err := notification.New(
	notification.WithChannel(mail.Name, mailChannel),
	notification.WithChannel(database.Name, databaseChannel),
)
if err != nil {
	return err
}
```

The manager rejects empty names, duplicate registrations, and nil channels.
Channel constructors reject nil sender/writer dependencies, including typed nils.
Custom channels implement `notification.Channel`; `notification.ChannelFunc`
adapts an inline function without a separate execution path.

### On-demand mail

`mail.To("Taylor <taylor@example.com>")` parses an anonymous recipient. Check its
error, then pass the recipient to `Send` with a mail notification. It provides a
mail route but no inbox identity, so selecting the database channel fails.
See the [on-demand example](mail/example_test.go).

`mail.Sender.Send` receives an `Envelope` with the destination, message, stable
delivery ID, and logical notification ID. It is called once per channel attempt.
A successful call normally means the provider accepted the mail, not that it
arrived or was read.

## Read an inbox

`database.NewChannel` requires only `database.Writer.Insert`; it does not require
query methods. `database.Inbox` is a separate interface for user-facing access.
`database/memory.New()` implements both and is useful for tests or process-local
use. Its data is neither durable nor shared between application instances.

Given an inbox `store` and an authenticated recipient `key`:

```go
page, err := store.List(ctx, key, database.ListOptions{
	UnreadOnly: true,
	Limit:      20,
})
if err != nil {
	return err
}
for _, record := range page.Records {
	if err := store.MarkRead(ctx, key, record.ID); err != nil {
		return err
	}
}
```

- Every operation is scoped by tenant, recipient type, and recipient ID. Derive
  that key from authenticated application state, not untrusted request fields.
- `CountUnread` counts unread records; `Delete` removes one scoped record.
- An absent or differently owned record returns `database.ErrNotFound` from
  `MarkRead` and `Delete`, without exposing another recipient's record.
- Repeated `MarkRead` preserves the first read timestamp.
- Lists sort by descending creation time and ID. Limit defaults to 20, with a
  maximum of 100. Use `Page.NextCursor` with the same recipient and unread filter
  to request another page. Separate pages are not a transactional snapshot.
- Inbox records are not mail delivery logs. Store the data needed to display
  historical notifications even when the original business record changes.

Writer implementations must atomically deduplicate matching delivery IDs and
return `database.ErrConflict` for a reused ID with different recipient,
notification ID, business type, or JSON bytes. Repeated matching inserts preserve
the original creation and read timestamps. The memory store also copies JSON
bytes and read-time pointers on input/output.

## Delivery results and cancellation

`Send` preflights inputs, options, channel selection and registrations before any
channel is invoked. These errors return a zero `Result`; they do not guarantee
that an application-provided `Via` or option has no side effects of its own.
Message and route validation happens inside each channel and can therefore
produce partial success.

Channels execute serially in `Via` order. Ordinary guard or channel errors do
not prevent later channels from running. Per-channel outcomes are:

| Status | Meaning |
| --- | --- |
| `sent` | The channel returned nil. |
| `skipped` | The notification's send guard returned false with no error. |
| `failed` | A guard or channel returned an error. |
| `not_attempted` | Context cancellation prevented the channel from starting. |

Each failed/not-attempted entry has a `*notification.DeliveryError` with its ID,
channel and original cause. `Send` joins these errors, preserving `errors.Is`
and `errors.As`. Context cancellation stops unstarted channels; running
implementations must observe the context themselves. A channel that returns nil
remains `sent` even if cancellation happened during that call.

Application panics are not recovered. Nil managers and nil contexts panic.

## Stable IDs and reliability boundaries

`Send` generates a random logical notification ID unless
`notification.WithID(id)` is supplied. A delivery ID is a deterministic SHA-256
digest of length-prefixed notification ID, tenant, recipient type/ID, and channel.
Reusing the same inputs produces the same delivery ID. Do not reuse a logical ID
for different notification data or different anonymous recipients.

Stable IDs do not automatically deduplicate external effects. Pass the envelope
ID to a provider's idempotency API when available. The database channel can
deduplicate through its Writer contract, but mail may still be sent more than
once after a timeout or a retry. A transport failure does not prove the provider
did not accept the request.

This version has no queue adapter or hidden asynchronous mode. Applications can
use existing Queue typed tasks with explicit, serializable business payloads and
construct notifications in the worker. Do not blindly retry a whole multi-channel
Send: use a channel-specific task/notification when independent retry is required.
Queue submission after a database commit is not atomic publication; reliable
business-state/message publication needs an application Outbox or equivalent.
