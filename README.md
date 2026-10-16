# Bucket Object Store

A small HTTP service for saving, reading, and deleting text. Every object has a
bucket name and an object ID. For example, `photos` is the bucket and `pic1` is
the object ID in this URL:

```text
/objects/photos/pic1
```

## Requirements

- Go 1.23 or newer
- `curl` for trying the example requests

## Start the service

Open a terminal in this project folder and run:

```sh
go run ./cmd/server
```

The service asks which port to use. Enter a number between `1` and `65535`:

```text
What port do you want to use? (e.g. 8080): 8080
You selected port 8080.
```

Keep this terminal open while using the service. Press `Ctrl+C` to stop it.
If you want to start the service from a script without a prompt, set `PORT`:

```sh
PORT=9090 go run ./cmd/server
```

In the examples below, replace `8080` with the port you selected, if different.
Run the `curl` commands in a second terminal.

## Save, read, and delete an object

### Save text (`PUT`)

Send the text in the request body. This saves `hello world` as object `pic1` in
the `photos` bucket:

```sh
curl -i -X PUT --data-binary 'hello world' http://localhost:8080/objects/photos/pic1
```

The response is `201 Created` with the object ID:

```json
{"id":"pic1"}
```

If you save different text later to the same bucket and object ID, it replaces
the old text.

### Read text (`GET`)

```sh
curl -i http://localhost:8080/objects/photos/pic1
```

If the object exists, the response is `200 OK` and the body contains its text:

```text
hello world
```

If the bucket and object ID do not exist, the response is `404 Not Found`.

### Delete an object (`DELETE`)

```sh
curl -i -X DELETE http://localhost:8080/objects/photos/pic1
```

The response is `200 OK` if the object was deleted, or `404 Not Found` if it
was not there.

## Notes

- Request bodies can be up to 10 MiB.
- Data is kept in memory and is lost when the service stops.
- `PUT` returns `201` for both a new object and an update to an existing ID.

## AI assistance

ChatGPT assisted with discussing the bucket-local de-duplication design, and drafting the implementation and tests. The implementation was checked with `go test ./...` and with live HTTP  equests for `PUT`, `GET`, `DELETE`, and `GET` after deletion (which returned `404`). The code was also formatted with `gofmt` and compiled with `go build`.
