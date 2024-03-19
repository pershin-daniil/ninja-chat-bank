## Development Manifest

### Use English

For comments and documentation we use English only.

### `%v`-wrapping by default

By default, try to wrap errors with `%v` (not `%w`) to avoid accidental abstraction leakage:

```go
return fmt.Errorf("create failed job: %v", err)
```
