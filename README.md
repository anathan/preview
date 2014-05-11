# Preview

> The work you do while you procrastinate is probably the work you should be doing for the rest of your life.

This is the `preview` project, it provides a service to create, cache and serve preview images for different types of files.

## Rendering

The primary rendering agent uses image magic to create resized images using the `convert` command. It supports the following file types:

* jpg
* jpeg
* png

# Configuration

The preview application uses JSON for configuration. It will attempt to load configuration from the following paths, in this order:

1. ./preview.config
2. ~/.preview.config
3. /etc/preview.config

Additionally, using the `-c` or `--config` command line arguments, a configuration file can be passed when starting the application.

    $ preview -c servera.config

The configuration object has the following top level sections:

* common
* http
* storage
* imageMagickRenderer
* simpleApi
* assetApi
* uploader
* downloader

The "common" group has the following keys:

* "nodeId" - The unique identifier of the preview instance.
* "placeholderBasePath" - The directory that contains placeholder image information.
* "placeholderGroups" - A map of grouped types of file types to groups used to determine the availability of file types when displaying placeholder images.

The "http" group has the following keys:

* "listen" -  The binding pattern for the HTTP interface.

The "storage" group has the following keys:

* "engine" - The storage engine to use to persist source assets and group assets.
* "cassandraNodes" - An array of strings representing cassandra nodes to interact with. Only available when the engine is "cassandra".
* "cassandraKeyspace" - The cassandra keyspace that queries are executed against. Only available when the engine is "cassandra".

The "imageMagickRenderer" group has the following keys:

* "enabled" - Used to determine if the image magick rendering agent should be started with the application.
* "count" - The number of agents to run concurrently.
* "supportedFileTypes" - A map of strings to integers representing the file types that are supported by the renderer and the max file size to render.

The "simpleApi" group has the following keys:

* "enabled" - If enabled, the simple API will be available with the "/api" base URL on the listen port.
* "edgeBaseUrl" - The base URL used when crafting links to renders and placeholders.

The "assetApi" group has the following keys:

* "basePath" - The directory that local renders are stored.

The "uploader" group has the following keys:

* "engine" - The engine to use when uploading rendered images.
* "s3Key" - The AWS key to use when uploading rendered images to S3. Only available when the engine is "s3".
* "s3Secret" - The AWS secret to use when uploading rendered images to S3. Only available when the engine is "s3".
* "s3Buckets" - A list of buckets used to distribute rendered images to when uploading rendered images to S3. Only available when the engine is "s3".
* "s3Host" - The host and base URL used to submit requests to when uploading rendered images to S3. Only available when the engine is "s3".
* "localBasePath" - The base directory to copy local renders to. Only available when the engine is "local".

The "downloader" group has the following keys:

* "basePath" - The directory that downloaded files are stored to.

## Default Configuration

By default, the application will use the following configuration json:

```json
{
   "common": {
      "placeholderBasePath":"BASEPATH",
      "placeholderGroups": {
         "image": ["jpg", "jpeg", "png", "gif"]
      },
      "nodeId": "E876F147E331"
   },
   "http":{
      "listen":":8080"
   },
   "storage":{
      "engine":"memory"
   },
   "imageMagickRenderer":{
      "enabled":true,
      "count":16,
      "supportedFileTypes":{
         "jpg":33554432,
         "jpeg":33554432,
         "png":33554432
      }
   },
   "simpleApi":{
      "enabled":true,
      "edgeBaseUrl":"http://localhost:8080"
   },
   "assetApi":{
      "basePath":"BASEPATH"
   },
   "uploader":{
      "engine":"local",
      "localBasePath":"BASEPATH"
   },
   "downloader":{
      "basePath":"BASEPATH"
   }
}
```

The `BASEPATH` set is the ".cache" directory in the current working directory when the executable is run.

# Usage

Through configuration different API resources and render agents can be toggle and configured.

## Simple API

By default, the simple API resources are enabled.

## Asset API

This API set serves generated assets based on the location of the generated asset.

* If the location is "local", it will attempt to load serve the file from disk.
* If the location is HTTP, it will attempt to redirect the file.
* If the location is S3, it will attempt to cache the file locally and serve it from the cache.

## Static API

By default, the static API resources are enabled.

This API set allows placeholder images to be served from the "/static/" base URL.

## Storage

By default, the "memory" storage system is enabled. All source asset and generated asset records are lost when the process is stopped when the "memory" storage engine is used.

Alternatively, the "cassandra" engine can be enabled to persist records to Cassandra. When enabled, one or more cassandra nodes must be configured and the keyspace configured.

```cql
CREATE KEYSPACE preview WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 3 };
USE preview;
CREATE TABLE IF NOT EXISTS generated_assets (id varchar, source varchar, status varchar, template_id varchar, message blob, PRIMARY KEY (id));
CREATE TABLE IF NOT EXISTS active_generated_assets (id varchar PRIMARY KEY);
CREATE TABLE IF NOT EXISTS waiting_generated_assets (id varchar, source varchar, template varchar, state varchar, PRIMARY KEY(template, source, id));
CREATE INDEX IF NOT EXISTS ON generated_assets (source);
CREATE INDEX IF NOT EXISTS ON generated_assets (status);
CREATE INDEX IF NOT EXISTS ON generated_assets (template_id);
CREATE TABLE IF NOT EXISTS source_assets (id varchar, type varchar, message blob, PRIMARY KEY (id, type));
CREATE INDEX IF NOT EXISTS ON source_assets (type);

```

## ImageMagick Render Agent

By default, the imagemagick render agent is enabled.

To support creating images for PDF files, the `gs` application in the ghostscript package is required.

## Uploader

By default, the "local" uploader is enabled. This uploader engine will simply copy rendered images from the temporary file/directory to the configured base path.

Alternatively, the "s3" engine can be enabled. With the key, secret, buckets and host set, rendered images will be uploaded to an S3 providing host.

## Downloader

The downloader cannot be disabled. The only configuration is the base directory in which files are downloaded from. It is important to understand how the downloader will attempt to count the number of references to a downloaded file. Once a file has been "released", temporary file manager will attempt to delete the file, freeing disk space.

## Running The Service

To run the service, execute the preview command.

    $ preview

# Contributing

1. Run `go fmt */*.go` before committing code.
2. Run the tests before pushing code to the code repository.
3. Run integration tests regularly.
4. Consider using [golint](https://github.com/golang/lint) before pushing code to the code repository.

## Unit Tests

Unit tests use the `testing` golang package.

When authoring unit tests, consider the amount of time taken to execute the test. If the test is particularly long (more than a second) then consider skipping the test when only short tests are run.

    $ go test ./...
    $ go test ./... -test.short

## Integration Tests

Integration tests are written the same way as unit tests but are skipped unless the -test.integration flag is present. Integration tests are assumed to require a development environment where Cassandra, Tram and s3ninja are running.

    $ go test ./... -test.integration -v

## Misc

Golint is a style correctness tool.

    $ go get github.com/golang/lint/golint
    $ golint .../*.go

Govet is a static code analyzer.

    $ go vet */*.go

To determine the size of the project:

    $ find . -type f -name '*.go' -exec wc -l {} \; | awk '{total += $1} END {print total}'
