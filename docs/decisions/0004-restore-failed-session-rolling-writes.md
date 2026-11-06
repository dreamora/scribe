# Restore rolling writes after a caught session failure

A failed mining session must not leave its rolling-file changes available for a later checkpoint commit. Restore changed rolling files to their pre-session bytes, remove rolling files created by that failed session, and keep the session pending. This chooses restoration over retaining partial rolling output; if restoration fails, stop mining and checkpoint commits. The guarantee covers caught failures, including model errors after partial writes; process-crash recovery is outside this fix.
