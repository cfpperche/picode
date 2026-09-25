### Fixed

- **Attach: Omp follow-ups keep their lines.** A follow-up sent to a working
  Omp arrives as one message with its line breaks, so a numbered list no
  longer splits into several follow-ups.
- **Attach: Hermes follow-ups say they are queued.** Hermes Agent shows a
  follow-up only when its turn ends; the composer now says so instead of
  warning that the message could not be confirmed.
- **Attach: Stop and send works on an Omp that has not answered yet.**
  Stopping Omp before its first output used to be reported as not stopped;
  PiCode now sees the stop and sends the message.
