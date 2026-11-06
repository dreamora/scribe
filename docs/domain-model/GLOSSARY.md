# Scribe session mining

Session mining extracts reusable knowledge from coding-agent sessions into a knowledge base.

## Language

**Rolling memory file**:
A Markdown file that accumulates lessons or decisions in a knowledge base. The standard filenames are `learnings.md` and `decisions-log.md`.

**Rolling memory archive**:
A rolling memory file that contains older entries moved from a live rolling memory file. Standard archive filenames include `-archive-YYYY` before `.md`.

**Project**:
A source repository enrolled for knowledge extraction. Its rolling memory files use its project directory under `projects/`.

**Domain**:
A label that classifies knowledge-base content. Configured domains and the universal `general` and `personal` domains are valid frontmatter values.

**Tools mode**:
The session-mining protocol in which `claude -p` reads sessions through ccrider MCP and writes knowledge-base files with filesystem tools.

**Envelope mode**:
The session-mining protocol in which Go reads the transcript and applies structured actions returned by the model.
