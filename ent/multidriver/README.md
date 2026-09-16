# Ent multi-driver

`multidriver.New` combines a writer and optional readers into an Ent `dialect.Driver`.
`Exec`, transactions, and queries without an Ent query context use the writer.
Queries with an Ent query context use the configured reader policy. Without readers,
reads also use the writer.

```go
driver, err := multidriver.New(
	multidriver.WithWriter(writer),
	multidriver.WithReaders(reader1, reader2),
)
```

## Resource ownership and closing

After successful construction, `Driver.Close` manages the supplied drivers:

- It closes the writer first, then readers in their first configured order.
- Equal, comparable driver values are closed once. For the usual pointer-based
  drivers, sharing the same pointer between writer/readers or repeating it in the
  reader list does not cause extra closes.
- Without explicit readers, the writer is closed once, including when its concrete
  driver value is not comparable.
- Non-comparable values cannot be deduplicated by interface equality. Each explicitly
  supplied entry is closed once; use a shared pointer when those entries represent
  one resource. Different wrappers are treated as different drivers, even if they
  internally share a connection.
- Repeated or concurrent calls to `Close` wait for the same closing operation and
  return its result. A failed close is not retried by calling `Close` again.

The reader slice is copied during construction. Changing the caller's slice later
does not replace the configured drivers. Duplicate reader entries remain in the
query policy's list, preserving their routing weight; only closing is deduplicated.

Stop issuing work before closing the multi-driver. Handling of in-flight operations
is delegated to each underlying driver. Closing one multi-driver does not coordinate
with another multi-driver that was given the same resource.

## Close errors

Every configured resource is given a chance to close even if an earlier close fails.
Success returns `nil`; failure joins `ErrClose` with all underlying errors.

```go
if err := driver.Close(); err != nil {
	if errors.Is(err, multidriver.ErrClose) {
		// Inspect or report the complete error, including the underlying causes.
		log.Print(err)
	}
}
```

Use `errors.Is(err, multidriver.ErrClose)` rather than `err == multidriver.ErrClose`.
Previously only the sentinel was returned; the joined error now preserves the
underlying causes for `errors.Is` and `errors.As`.
