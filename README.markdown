# urlwatch

This is a simple commandline utility to watch a web page that's not
working and send me some notifications to let me know when it starts
working again.

Currently it supports growl and webhooks.  Could do more in the future.

## Howto

### Step one

Create a notify.json from the example, but with your stuff in it.  It
can have as many or as few notifiers as you want (if it's empty, the
process will just exit when it's done).

### Step two

run the tool:

    ./urlwatch http://www.google.com/
