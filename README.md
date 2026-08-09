# AT Mail

A simple SMTP and IMAP4 server.

## Goals

Easy to deploy, to configure and to use, because connecting an SMTP server and an IMAP server is never straightforward.
Integrates well with rspamd to sign outgoing emails (via DKIM) and to filter unwanted emails.

Encrypt emails with user's GPG public key to avoid storing sensitive information.

Custom storage backend to be fast and lightweigth.
Automatically compress big emails with gzip.

Use one sqlite3 database per user to avoid using a centralized database.

## Usage

`atmail` reads the configuration at `/etc/atmail/config.toml`.
You can override it with `-config <path>`.
If the configuration is not found, it creates a new default configuration.

If you want to send logs to syslog MAIL facility, add `-syslog`.

If you want to increase the verbosity, add `-v`.

### Common config

```toml
# defines the path to the global DB (currently unused, may be removed later)
db = "/var/lib/atmail/data.db"
# directory containing the mails
directory = "/var/lib/atmail/data"
# folder containing domains' config file
domains_folder = "/etc/atmail/domains.d/"

# domain broadcasted in SMTP HELO/EHLO
main_domain = "mail.example.org"
```

### SMTP and IMAP

```toml
[smtp]
# address to listen to (tcp or unix socket)
listen = ":25"
# use PROXY V2 Protocol by HAProxy
use_proxy_v2 = false
# max mail size in KiB: a single email cannot exceed this.
max_mail_size = 16416
```

```toml
[imap]
listen = ":993"
use_proxy_v2 = false
```

### rspamd

```toml
[rspamd]
# Address of the rspamd server
#address = "localhost:11332"
```

### Domains

AT Mail can handle multiple domains.
There is three kind of domain:
- static;
- catch all;
- ATProto (not implemented yet).

A kind defines how users are managed by the server.

**Static** defines users in the config file and links them with their password hashed with bcrypt:
```toml
[domains."example.org".static.users]
foo.password = "$2y$10$wWJy64n5j4pU2CscdWRq.er5Z2.V31U5Ntz5uTopLy1fMX87GQEDG"
#foo.pgp_pub_key = ""
#foo.pgp_pub_key_file = ""

bar.password = "$2y$10$fZ1tRpz2mfdRfaaTbx723.Gch7tY.f7LIBuZz1of/cLwBGvU39ivG"
#bar.pgp_pub_key = ""
#bar.pgp_pub_key_file = ""
```
You can also limit an account to be used by only local IP address regardless of their password:
```toml
baz.local_only = true
```

**Catch All** redirects every emails to one address:
```toml
[domains."example.org".catch_all]
user = "foo"
password = "$2y$10$fZ1tRpz2mfdRfaaTbx723.Gch7tY.f7LIBuZz1of/cLwBGvU39ivG"
#pgp_pub_key = ""
#pgp_pub_key_file = ""
```
With this configuration, an email to `bar@example.org` is redirected to `foo@example.org`.

**ATProto** links an ATProto PDS account with an email account.
This kind is not implemented yet.
```toml
[domains."example.org".atproto]
pds = "pds.example.org"
client_id = "mail.example.org"
client_secret = "my-secret"
```

#### Subaddressing

You can enable the plus subaddressing (e.g., `foo+bar@example.org` is redirected to `foo@example.org`) for one domain
with:
```toml
[domains."example.org"]
plus_subaddressing = true
```
This option is always enabled if the domain is a catch all.

You can also automatically create one IMAP mailbox (usually called folder) per subaddress:
```toml
[domains."example.org"]
create_folder_subaddressing = true
```
If enabled, an email to `foo+bar@example.org` will be placed in `INBOX/bar`.
If the domain is a catch all one, every redirected email is placed in `INBOX/address` (e.g., `foo@example.org` goes in
`INBOX/foo`).

#### Admin

Each domain should have an administrator account.
To promote a regular account into an administrator one, add a section `admin` to the domain:
```toml
[domains."example.org".admin]
# username of the account
user = "admin"
# IMAP folder to use
#folder = ""
# if you wanna encrypt with PGP
#pgp_pub_key = ""
#pgp_pub_key_file = ""
```

This account can be reached with `postmaster`, `root`, `operator`, `security` and `abuse` usernames.
These are reserved and cannot be used for a regular user.

### Splitting config file

Each domain can have its own config file.
They are placed in `/etc/atmail/domains.d/` by default.
Their name must follow this scheme: `domain.tld.toml`, where `domain.tld` is the domain.
This configuration always overrides the one defined in the main config file.

The file contains the complete definition of the domain.
It is directly injected in the section `[domains."domain.tld"]`, thus you must omit this prefix, e.g.,
```toml
plus_subaddressing = true

[admin]
user = "noreply"

[static.users]
noreply.password = "$2y$10$wWJy64n5j4pU2CscdWRq.er5Z2.V31U5Ntz5uTopLy1fMX87GQEDG"
foo.password = "$2y$10$wWJy64n5j4pU2CscdWRq.er5Z2.V31U5Ntz5uTopLy1fMX87GQEDG"
```

## Roadmap

Planned features:
- QUOTA IMAP capability
- better PGP integration

Features considered:
- JMAP support
