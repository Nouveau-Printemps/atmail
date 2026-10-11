# Filter DSL

Filter is a **DSL** (domain specific language) created to write human-readable filters.
```filter
from = "me@example.org" or from.domain = "example.net"
to.domain = "example.com"

let object = header:get("Object")

filter "free things spam" do
	object:contains("FREE")

	do spam()
end

filter "AI bullshit" do
	object:contains("AI") or body:contains("AI") or body:contains("LLM")

	do mailbox("AI")
	do quarantine()
end
```

## Principe

There are 4 statements:
- `let` which defines a variable
- `filter` which creates a sub-filter
- `do` which performs an action
- every other is a condition

Each variable is immutable and each execution is deterministic: a filter always returns the same result for the same
data.

## Conditions

The statement
```filter
from = "me@example.org"
```
is a condition that is met when the `from` variable equals the string `me@example.org`.
If `from` is not equal to this string, the rest of the filter is not executed.

The common operators `=`, `!=`, `>=`, `<=`, `>` and `<` are supported.
The boolean OR operator is `or`.
Because each condition must be true, there is no "and" operator.

The following C-like conditions
```c
if (spam_score < 10 && message != "spam") return;
if (custom == "no_spam") return;

// or
if ((spam_score >= 10 || message == "spam") && custom != "no_spam")
```
can be translated into
```filter
spam_score >= 10 or message = "spam"
custom != "no_spam"
```
The conditions are inverted because they must be true to be executed!

> [!NOTE]
> It is easier to understand: write your hypothesis instead of thinking about why it shouldn't be applied.

Some predefined variables have fields.
You can access them with `variable.field`, e.g.
```filter
to.domain = "example.com"
```

Some types also have methods that can be called like `variable:method(param1, param2)`, e.g.
```filter
to:contains("@")
spam_score:number()
```

## Defining variables

You can save a value in a variable with
```filter
let name = value
```
e.g.
```filter
let true = 0 = 0
```

The next statement will always be executed.

## Performing actions

You can perform an action with side effects with
```filter
do action(param1, param2)
```
e.g.
```filter
do forward("postmaster@example.org")
do forward("abuse@example.org")
do spam()
```

Some actions are terminal: they stop the execution and returns a specific action to perform.
If the it is not, the next statement will always be executed.

## Creating sub-filters

It can be repetitive to express always the same conditions at the begining.
You can create sub-filters that are executed when the above conditions are met.
```filter
filter "name" do
	# your code
end
# or
filter do
	# your code
end
```
e.g.
```
from.domain = "example.org"

filter "to example.com" do
	to.domain = "example.com"

	do forward("dotcom@example.org")
	do accept()
end

filter "to example" do
	to.domain:contains("example")

	filter "postmaster" do
		to.user = "postmaster"

		do forward("postmaster@example.org")
		do reject()
	end

	do quarantine()
end
```

If the filter executes a terminal action, other statements (including those outside) are not executed.
Otherwise it doesn't stop the execution.
