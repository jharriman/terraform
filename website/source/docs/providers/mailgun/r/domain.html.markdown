---
layout: "mailgun"
page_title: "Mailgun: mailgun_domain"
sidebar_current: "docs-mailgun-resource-domain"
---

# mailgun\_domain

Provides a Mailgun App resource. This can be used to
create and manage applications on Mailgun.

## Example Usage

```
# Create a new mailgun domain
resource "mailgun_domain" "default" {
    name = "test.example.com"
    spam_action = "disabled"
    smtp_password = "foobar"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The domain to add to Mailgun
* `smtp_password` - (Required) Password for SMTP authentication
* `spam_action` - (Optional) `disabled` or `tag` Disable, no spam
    filtering will occur for inbound messages. Tag, messages
    will be tagged wtih a spam header.
* `wildcard` - (Optional) Boolean determines whether
    the domain will accept email for sub-domains.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the domain.
* `smtp_login` - The login email for the SMTP server.
* `smtp_password` - The password to the SMTP server.
* `wildcard` - Whether or not the domain will accept email for sub-domains.
* `spam_action` - The spam filtering setting.
