# Configuration example
```toml
debug = true
ignore_suffixes = ["ignore1.example.com", "ignore2.example.com"]

[nats]
url = "nats://nats:4222"
event_subject = "internal.events.new_qname"
observation_subject_prefix = "internal.observations"
private_subject_prefix = "internal.service.tapir-analyse-new-qname"
seen_domains_subject_prefix = "internal.seen-domains"
globally_new_bucket = "globally_new"
private_bucket = "tapir-analyse-new-qname"
seen_domains_bucket = "seen_domains"

[cert]
active = false

[api]
active = false

[libtapir]
debug = true
```
