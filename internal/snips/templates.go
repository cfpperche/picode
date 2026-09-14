// Package snips: built-in starters (snippets v2, F6). The list is code,
// not data, like internal/automate and internal/slashres: it ships with
// the binary, needs no migration, and a body that stops parsing fails a
// unit test instead of a customer.
package snips

// Template is a suggested snippet. Served by GET /api/snips/templates
// and pre-filled into the editor — never created directly, so the reader
// can change anything before it is saved.
type Template struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Kind        string   `json:"kind"` // prompt | shell
	Tags        []string `json:"tags"`
	Body        string   `json:"body"`
}

// Templates returns the built-in starters in display order: the six
// things a reader most often wants a reusable prompt for, each one a
// shape they can edit rather than a script they have to trust.
func Templates() []Template {
	return []Template{
		{
			ID: "review-pr", Title: "Review a pull request",
			Description: "Read a diff and report what it does and what it risks.",
			Kind:        "prompt", Tags: []string{"review", "git"},
			Body: "Review the changes in {{ref=the current branch}} against {{base=main}}.\n\n" +
				"Read the diff first (`git diff {{base}}...{{ref}}`). Report, in this order:\n" +
				"- what the change does, in two sentences\n" +
				"- correctness risks, each with file and line\n" +
				"- behaviour that changed without a test\n" +
				"- one thing you would do differently\n\n" +
				"Do not modify anything.",
		},
		{
			ID: "explain-error", Title: "Explain an error",
			Description: "Cause first, then the fix, for one pasted error.",
			Kind:        "prompt", Tags: []string{"debug"},
			Body: "Explain this error and how to fix it:\n\n{{error}}\n\n" +
				"One paragraph on the cause, then the fix as a short diff or one or two commands. " +
				"If the message alone is ambiguous, say what you would need in order to be sure.",
		},
		{
			ID: "commit-message", Title: "Write a commit message",
			Description: "A message in the house style for what is staged.",
			Kind:        "prompt", Tags: []string{"git", "commit"},
			Body: "Write a commit message for the staged changes in {{path=.}}.\n\n" +
				"Read `git diff --staged` and `git log --oneline -10` for the house style. " +
				"Subject line imperative and under 72 characters, then a body saying what changed and why. " +
				"Output the message only — do not commit.",
		},
		{
			ID: "summarize-changes", Title: "Summarize uncommitted changes",
			Description: "What is in the working tree, grouped by purpose.",
			Kind:        "prompt", Tags: []string{"git"},
			Body: "Summarize the uncommitted work in {{path=.}} (`git status --short` and `git diff`).\n\n" +
				"Group the changes by purpose, name the files, and call out anything that looks accidental " +
				"or half-finished. End with the one thing to check before committing. Do not modify anything.",
		},
		{
			ID: "run-tests-fix", Title: "Run the tests and fix failures",
			Description: "Full suite, then fix what is clearly a bug.",
			Kind:        "prompt", Tags: []string{"test", "fix"},
			Body: "Find this project's test command and run the whole suite.\n\n" +
				"For every failure: the test name, the error, and the most likely cause with file and line. " +
				"Then fix the failures that are clearly bugs in the code under test, and re-run until it is " +
				"green or a failure needs a decision — say which, and why. Never change a test to make it pass.",
		},
		{
			ID: "standup-update", Title: "Standup update",
			Description: "Yesterday, today, blocked — from real commits.",
			Kind:        "prompt", Tags: []string{"report"},
			Body: "Write my standup update from what actually happened in this repository " +
				"(commits since {{since=yesterday morning}}, branches with recent activity, uncommitted work).\n\n" +
				"Three short sections — done, doing, blocked — as bullets, no filler, and nothing that has " +
				"no commit or diff behind it.",
		},
	}
}
