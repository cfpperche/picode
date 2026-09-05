package main

// A user's Quit is successful even if setup failed before the menu appeared.
// The scheduler may retry failures, but must not undo an explicit user exit.
func trayResult(setupErr error, quitRequested bool) error {
	if quitRequested {
		return nil
	}
	return setupErr
}
