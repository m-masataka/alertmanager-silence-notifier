Notify Added/Expired Silence.

# Usage
build

```
go build notifier.go
```

run

```
./notifier \ 
 --alertmanager.host=localhost \
 --alertmanager.port=9093 \
 --slack.username=Bot \
 --slack.channel=general \
 --slack.token={your webhook token}
 --interval=10s \
 --timerange=5m
```

