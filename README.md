# jsonx

A tiny Go package that makes JSON decoding **actually tell you what went wrong**.

```
go get github.com/v8tix/jsonx
```

---

## Why?

The standard library's `encoding/json` is great, but its errors can be... terse.

```
invalid character '}' looking for beginning of value
```

`jsonx` gives you this instead:

```
badly-formed JSON in the body: line 3 col 11 (near "\"age\": }")
```

Every error wraps a typed sentinel so you can branch on it with `errors.Is`,
and every message includes the line, column, and a context snippet so you can
fix the problem without guessing.

---

## Quick start

```go
import "github.com/v8tix/jsonx"

type Order struct {
    ID    string `json:"id"`
    Total int    `json:"total"`
}

func parseOrder(r io.Reader) (Order, error) {
    return jsonx.ReadJSONAs[Order](r)
}
```

That's it. `ReadJSONAs` is generic — you get a fully-typed value back, no
casting required.

---

## What it checks

| Situation | Sentinel | Message includes |
|-----------|----------|-----------------|
| Empty body | `ErrEmptyBody` | — |
| Syntax error | `ErrBadlyJSON` | line, col, context snippet |
| Truncated JSON | `ErrBadlyJSON` | line, col at cut point |
| Wrong type for a field | `ErrBadJSONType` | field name, expected Go type, JSON token |
| Unknown field in object | `ErrBodyUnknownKey` | the offending key name |
| More than one top-level value | `ErrBodyValue` | — |
| Body too large | `ErrBodySizeLimit` | the configured byte limit |

All errors wrap their sentinel, so `errors.Is` always works — even through
additional wrapping layers:

```go
val, err := jsonx.ReadJSONAs[Order](r)
switch {
case err == nil:
    // happy path
case errors.Is(err, jsonx.ErrBadlyJSON):
    http.Error(w, "malformed JSON: "+err.Error(), http.StatusBadRequest)
case errors.Is(err, jsonx.ErrBadJSONType):
    http.Error(w, "wrong value type: "+err.Error(), http.StatusUnprocessableEntity)
case errors.Is(err, jsonx.ErrBodyUnknownKey):
    http.Error(w, "unexpected field: "+err.Error(), http.StatusBadRequest)
case errors.Is(err, jsonx.ErrEmptyBody):
    http.Error(w, "request body is required", http.StatusBadRequest)
default:
    http.Error(w, "internal error", http.StatusInternalServerError)
}
```

---

## Body size limit

`ReadJSONAs` buffers the full body in memory. If you need to cap allocation,
wrap the reader with [`http.MaxBytesReader`](https://pkg.go.dev/net/http#MaxBytesReader)
before passing it in — `jsonx` will translate the limit error into `ErrBodySizeLimit`:

```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB cap
order, err := jsonx.ReadJSONAs[Order](r.Body)
if errors.Is(err, jsonx.ErrBodySizeLimit) {
    http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
}
```

---

## Strict decoding

`ReadJSONAs` always uses **strict** semantics:

- Unknown fields are rejected (`ErrBodyUnknownKey`) — no silent data loss.
- Trailing content after the first value is rejected (`ErrBodyValue`) — avoids
  accidentally accepting concatenated payloads.

---

## Notes

- Line and column numbers count **bytes**, not Unicode runes — consistent with
  the offsets reported by `encoding/json`.
- The context snippet around a syntax error is ±20 bytes, `%q`-formatted.

---

## License

[MIT](./LICENSE) — free to use, modify, and ship in commercial products.
