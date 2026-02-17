# alnvdl/anything

Anything is a web application for two or more people to vote on what's for
breakfast, lunch or dinner. See the README.md for a more detailed description
of how it works.

## Security
While having a secret as a URL parameter is usually not OK, it's considered a
reasonable trade-off in this system that can only be used to edit entries and
submit votes for those entries.

Never add outbound links from this application. Since the token is embedded in
the URL, that would mean exposing the token in the Referer header.

## Coding
Never introduce new external dependencies, unless explicitly told to do so.

You are free to use anything from the standard library of any language.

You are free to use imports that are already present in the files you are
working with.

Always end comments with a period in all languages.

When declaring slices in Go test code, make sure the curly braces are placed as
compactly as possible. For example:
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

The description for each Go subtest should be named `desc` and it should always
start with a number or lowercase letter.
