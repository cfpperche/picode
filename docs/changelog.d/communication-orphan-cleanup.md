### Fixed
- **Private connection setup no longer lingers on disk.** PiCode removes a conversation's private setup files once its connection is revoked or the owner is deleted; the credential stopped working before, but the 0600 files used to stay behind.
