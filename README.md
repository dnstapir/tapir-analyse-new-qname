# Configuration example
```toml
debug = true
ignore_suffixes = ["ignore1.example.com", "ignore2.example.org"]

[nats]
debug = true
url = "nats://nats:4222"
event_subject = "internal.events.new_qname"
observation_subject_prefix = "internal.observations"
seen_domains_subject_prefix = "internal.seen-domains"
private_subject_prefix = "internal.service.tapir-analyse-new-qname" # Currently unused

[[nats.observation_buckets]]
observation = "globally_new"
name = "globally_new"
create = false # A "false" setting requires bucket to be pre-provisioned
ttl = 3600

[nats.seen_domains_bucket]
name = "seen_domains"
create = true # A "true" setting does not require a bucket to be pre-provisioned
# Not setting ttl will let values live in bucket indefinitely

[api]
active = false
```
