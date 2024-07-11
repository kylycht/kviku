# In Memory Cache

## Running application as standalone service

If you intend to run application as standalone service, you can run master node via:

```bash
cd cmd/master
go run main.go --listenAddr :8080
```

### Running application w/ replica node

```bash
cd cmd/slave
go run main.go --listenAddr :8888
```

In another terminal issue

```bash
cd cmd/master
go run main.go --listenAddr :8080 --slaveAddr=0.0.0.0:8888
```

#### Endpoints

1. POST /store?key=1&value=a&ttl=10m (also expires_at is supported in RFC3339Nano format, omitted ttl/expires_at will never expire!)
2. GET /load?key=1

##### NOTES

Few critical items are omitted being a demo task:

1. Proper unit tests
2. Proper logging
3. Makefile
4. Full README file
