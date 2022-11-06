Official repository and project is on [codeberg](https://codeberg.org/bacardi55/ggm). Github is only a mirror.

# Go Gemini Mention

[![builds.sr.ht status](https://builds.sr.ht/~bacardi55/ggm.svg)](https://builds.sr.ht/~bacardi55/ggm?)
[![license: AGPL-3.0-only](https://img.shields.io/badge/license-AGPL--3.0--only-informational.svg)](LICENSE)

Official repository and project is on [sourcehut](https://git.sr.ht/~bacardi55/ggm). Github and codeberg are only mirrors.


This is a small program to manage Gemini mention.

WARNING: This is a very beta program to implement the gemini mention RFC, use with caution!

To learn about gemini mention, please see [this page](https://codeberg.org/bacardi55/gemini-mentions-rfc).

## Installation

Either download the source and run `make dependencies && make build`, then upload the binary in `./bin/` to your capsule at the `/.well-known/mention` path (so you need to rename the binary to `mention`).
It must be executable (`chmod +x mention`).

Or you can download binaries on the release page.

## Configuration

You need to create a configuration file at `/etc/gogeminimention.toml`.

If you wish to use another path, you must configure a environment variable `GGM_CONFIG_PATH` with the path to your file. This is because a CGI script will not be able to start the program with a cli argument (to override config path).

The config file should be as follow:

```toml
# Configuration file for Go Gemini Mention (aka ggm).
# This file should be kept outside of your capsule root to avoid access to it!

# Global configuration
# Capsule root address, without the "gemini://" URL scheme.
# Example: "gmi.bacardi55.io" or "mydomain.com/~user"
capsuleRootAddress = "mydomain.net"
# The maximum number of mentions you would like to see in the notification.
# This will limit the number of requests you send to your own capsule.
# Normally it should be one, but a limit prevent abuse :).
maxMentions = 2
# The email address you want to receive the notification to.
contact = "user@example.com"

# The access to the log file to use:
log = "/tmp/ggm.log"

# Notification configuration
# For now, you need to indicate in clear the login/password.
# It isn't very secure but this is the first pass at this tool :)
# I strongly suggest that you use a dedicated email account for this just in case.
# If gemini mention start to be more used, I'll add a better way.

# The address of the smtp server:
smtpServer = "mail.example.net"
# The port of the smtp server:
port = 587
# The email of the sender (from field):
from = "sender@example.net"
# The login/user (often the email address)
login = ""
# The password for the above user:
password = ""
```

See an [example](/ggm.toml.example).
