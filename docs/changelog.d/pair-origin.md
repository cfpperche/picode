### Security

- **A page open in another tab can no longer submit pairing codes to your
  PiCode.** The pairing form was the one door left outside the cross-site
  check, so another site could post guesses at it — it still needed a code
  off your screen, and five wrong ones lock it out, but there was no reason
  to leave the door swinging. Reading the pairing page is unchanged, and so
  is pairing itself: `picode pair` and any script still work, and a device
  arriving on an address PiCode has never seen can still pair, because the
  address check deliberately stays off this route.
