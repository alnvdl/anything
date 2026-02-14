# alnvdl/anything

Anything is a web application for two or more people to vote on what's for
breakfast, lunch or dinner.

## Coding
Never introduce new external dependencies, unless explicitly told to do so.

You are free to use anything from the standard library of any language.

You are free to use imports that are already present in the files you are working with.

Always end comments with a period in all languages.

When declaring slices in Go test code, make sure the curly braces are placed as compactly as possible. For example:
```go
var tests = []struct {
	desc string
}{{
	desc: "...",
}, {
	desc: "...",
}}
```

Prefer table tests, unless instructed otherwise.

Use the `test` variable when iterating over subtests in a Go table test.

The description for each Go subtest should be named `desc` and it should always start with a number or lowercase letter.
