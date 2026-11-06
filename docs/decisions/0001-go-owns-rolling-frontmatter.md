# Go owns rolling-file frontmatter validity

Session mining must use Go to guarantee valid rolling-file frontmatter before it records a session as processed. Model instructions alone cannot enforce the required metadata, so they are insufficient as the fix for [issue #2](https://github.com/dreamora/scribe/issues/2). This decision preserves the file locations that tools mode and envelope mode currently use; it does not select a repair algorithm or authorize implementation.
