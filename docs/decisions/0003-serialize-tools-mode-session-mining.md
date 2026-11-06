# Serialize tools-mode session mining

Tools-mode sessions run serially so Go can validate each session's rolling-file output before it records success. Separate model processes can write the same file, so an in-process mutex cannot protect their filesystem operations. This reduces tools-mode throughput but avoids the larger change needed to isolate and merge each session's output; envelope-mode parallelism remains supported.

Envelope-mode model calls remain parallel, but Go serializes each session's apply, validation, and restoration phase. Checkpoints run only between these phases. This prevents a failed session's restoration from undoing another session's successful writes or a checkpoint from capturing unvalidated output.
