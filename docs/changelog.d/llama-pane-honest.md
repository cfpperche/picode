### Fixed
- The llama.cpp page checks a server that does not answer in about 4 seconds instead of 18, and **Save connection** is no longer blocked while it checks.
- Connection problems now say what they are — nothing listening, name not found, untrusted certificate, no answer in time — instead of one "Cannot reach the server" for all of them.
- The Models tab says why a configured server cannot be managed, and picking a Hugging Face repo shows that it is being read; a slower earlier pick no longer replaces a later one.
- A quantization whose size Hugging Face does not report is listed instead of dropped (the recommended Q4_K_M could vanish).
- Adding llama.cpp no longer says "Signed in" before anything was checked.
- The download dialog's quantization rows no longer overlap; each row's Download button lines up on the right.
