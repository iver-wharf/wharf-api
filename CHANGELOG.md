# Wharf API changelog

This project tries to follow [SemVer 2.0.0](https://semver.org/).

<!--
	When composing new changes to this list, try to follow convention.

	The WIP release shall be updated just before adding the Git tag.
	From (WIP) to (YYYY-MM-DD), ex: (2021-02-09) for 9th of Febuary, 2021

	A good source on conventions can be found here:
	https://changelog.md/
-->

## v5.0.0 (WIP)

- BREAKING: Removed all deprecated environment variable configs, which were
  marked as deprecated in v4.2.0/#38. Now all environment variables require the
  `WHARF_` prefix. (#87)

- BREAKING: Changed the following POST creation endpoints to solely create,
  instead of the previous behavior where it instead could update if it found an
  existing database object that matched the HTTP request data: (#88, #93)

  - `POST /project`
  - `POST /provider`
  - `POST /token`

- BREAKING: Removed RabbitMQ integration. All `mq.*` YAML configs and
  `WHARF_MQ_*` environment variables are no longer relevant. This may be
  implemented again later, but inside a new "notification" component instead of
  directly inside wharf-api. (#102)

- Deprecated PUT endpoints that took the object ID from the HTTP request body.
  They are still supported, but may be removed in the next major release
  (v6.0.0). Please refer to the new endpoints that takes the ID from the URL
  path. (#88, #91, #94, #97)

  - Use `PUT /project/{projectId}` instead of `PUT /project`
  - Use `PUT /provider/{providerId}` instead of `PUT /provider`
  - Use `PUT /token/{tokenId}` instead of `PUT /token`
  - Use `PUT /project/{projectId}/branch` instead of `PUT /branches`

- Added support for Sqlite. Default database driver is still Postgres.

  Note: wharf-api must be compiled with `CGO_ENABLED=1` (which is the default
  for Go builds) but our Docker build is compiled with `CGO_ENABLED=0`. If you
  need Sqlite support in our Docker image, then please file a new issue over
  at <https://github.com/iver-wharf/wharf-api/issues/new>, and we will take a
  look at it. (#86)

- Added configuration for selecting database driver, environment variable
  `WHARF_DB_DRIVER` or the YAML key `db.driver`. Valid values: (#86)

  - `postgres` (default)
  - `sqlite`

- Added configuration for Sqlite file path, environment variable `WHARF_DB_PATH`
  or the YAML key `db.path`. Defaults to `wharf-api.db`. (#86)

- Added dependency on `gorm.io/driver/sqlite`. (#86)

- Fixed bug where unable to delete a Project without first deleting all child
  objects. (#64)

- Fixed where wharf-core logging for Gin debug and error messages were set up
  after they were initially used, leading to a mix of wharf-core and Gin
  formatted logs. (#63)

- Added database tables: (#43)

  - test_result_detail
  - test_result_summary

- Added test-result specific endpoints: (#43)

  - `POST /build/{buildid}/test-result`

    This should be used instead of `POST /build/{buildid}/artifact`
    when uploading test result files.

  - `GET /build/{buildid}/test-result/detail`

  - `GET /build/{buildid}/test-result/summary`

  - `GET /build/{buildid}/test-result/summary/{artifactId}`

  - `GET /build/{buildid}/test-result/summary/{artifactId}/detail`

  - `GET /build/{buildid}/test-result/list-summary`

- Deprecated endpoint `GET /build/{buildid}/tests-results`.

  Use `GET /build/{buildid}/test-result/list-summary` instead. The response
  data is slightly different; it has additional properties, and does not have a
  `status` property. (#43, #77)

- Changed format of all endpoint's path parameters from all lowercase to
  camelCase: (#76)

  - branchid -> branchId
  - projectid -> projectId
  - providerid -> providerId
  - tokenid -> tokenId
  - buildid -> buildId

  This affects the Swagger documentation, but has no behavioral implications.

- Deprecated endpoint `GET /branch/{branchId}`. Getting a single branch by its
  ID has not been shown to have any benefits. Please refer to the
  `GET /project/{projectId}` endpoint instead. (#75)

- Removed `Provider.UploadURL` and all references to it, as it was unused. (#82)

- Removed DB column `provider.upload_url`, as it was unused. (#82)

- Added `TestResultListSummary` field to `Build` database model. This allows you
  to avoid `N+1` HTTP requests when listing builds to show test summaries. (#80)

- Changed to preload `TestResultSummaries` field of `Build` database
  model. (#80)

- Added packages for "Plain Old Go Objects", with finer-grained decoupling
  between database, HTTP request, and HTTP response models.
  The Swagger documentation is affected by this, and some unused fields have
  been removed from certain endpoints, such as the `tokenId` in `POST /token`.
  The new packages are: (#78, #83)

  - `pkg/model/database`
  - `pkg/model/request`
  - `pkg/model/response`

- Added more backend validation on some endpoints, such as enforcing `name`
  field to be set when creating a new project. (#83)

- Fixed `PUT /token` where it did not use the `providerId` value from the HTTP
  request body. It now sets the provider's token if the field is supplied and
  non-zero. (#78)

- Added Swagger operation IDs to all endpoints. This has no effect on the API's
  behavior, but affects code generators. (#79)

- Fixed bug where projects created using the deprecated `PUT /project` endpoint
  would have set a null `ProviderID` in the database. (#96)

- Added Swagger attribute `minimum` to all ID path parameters, response bodies,
  and request bodies, as we do not support negative values there. (#98)

- Changed a lot of database columns to be `NOT NULL` where wharf-api already
  didn't support null/nil values. Migration steps have been added so any
  potential null values will be changed to empty strings or zeroes.
  The updated columns are: (#100)

  - `artifact.file_name`
  - `build_param.value`
  - `param.default_value`
  - `param.value`
  - `project.avatar_url`
  - `project.build_definition`
  - `project.description`
  - `project.git_url`
  - `project.group_name`
  - `test_result_summary.file_name`
  - `token.user_name`

## v4.2.0 (2021-09-10)

- Added support for the TZ environment variable (setting timezones ex.
  `"Europe/Stockholm"`) through the tzdata package. (#40)

- Added config loading from YAML files using
  `github.com/iver-wharf/wharf-core/pkg/config` together with new config models
  for configuring wharf-api. See `config.go` or the reference documentation on
  the `Config` type for information on how to configure wharf-api. (#38, #51)

- Deprecated all environment variable configs. They are still supported, but may
  be removed in the next major release (v5.0.0). Please refer to the new config
  schema seen in `config.go` and `config_example_test.go`. (#38)

- Added Makefile to simplify building and developing the project locally.
  (#41, #42)

- Added wharf-core logging for Gin debug and errors logging. (#45)

- Added wharf-core logging for GORM debug logging. (#45)

- Changed version of `github.com/iver-wharf/wharf-core` from v1.0.0 to v1.2.0.
  (#45, #52)

- Added documentation to the remaining types in the project. No more linting
  errors! (#46, #54)

- Added new endpoints `/api/ping` and `/api/health`. (#44)

- Deprecated `/` and `/health` endpoints, soon to be moved to `/api/ping` and
  `/api/health` respectively, so they are aligned with the current Swagger
  documentation. (#44)

- Changed logging and moved the `httputils` package to stay consistent with the
  provider API repos. (#47)

- Changed version of `github.com/swaggo/swag/cmd/swag` from v1.7.0 to v1.7.1 in
  Dockerfile and Makefile. (#48)

- Changed logging on "attempting to reach database" during initialization from
  "ERROR" to "WARN", and rephrased it a little. (#50)

- Fixed so failed parsing of build status in the `PUT /build/{buildid}` and
  `POST /build/{buildid}/log` endpoints not silently ignore it and fallback to
  "Scheduling", but instead respond with appropriate problem responses. (#54)

- Removed constraint that project groups cannot be changed in the
  `PUT /project` endpoint. This deprecates the problem
  `/prob/api/project/cannot-change-group`. (#55)

- Removed dead/unused function `getProjectGroupFromGitURL` and type
  `logBroadcaster`. (#57)

- Removed `internal/httputils`, which was moved to
  `github.com/iver-wharf/wharf-core/pkg/cacertutil`. (#52)

- Changed version of Docker base images, relying on "latest" patch version:

  - Alpine: 3.14.0 -> 3.14 (#59)
  - Golang: 1.16.5 -> 1.16 (#59)

## v4.1.1 (2021-07-12)

- Changed version of Docker base images:

  - Alpine: 3.13.4 -> 3.14.0 (#31, #36)
  - Golang: 1.13.4 -> 1.16.5 (#36)

- Changed version of GORMv2 from v1.20.12 to v1.21.11. No major changes, but as
  a bug in GORM was fixed we could finally move to the latest version. (#34)

- Changed references to wharf-core pkg/problem and pkg/ginutil. (#33)

- Changed all logging via `fmt.Print` and sirupsen/logrus to instead use the new
  `github.com/iver-wharf/wharf-core/pkg/logger`. (#37)

- Removed dependency on `github.com/sirupsen/logrus`. (#37)

## v4.1.0 (2021-06-10)

- Added endpoint `PUT /provider` as an idempotent way of creating a
  provider. (#28)

- Add endpoint `PUT /token` as an idempotent way of creating a
  token. (#26)

- Added environment var for setting bind address and port. (#29)

- Fixed missing `failed` field from `main.TestsResults` in
  `GET /build/{buildid}/tests-results`. (#25)

## v4.0.0 (2021-05-28)

- Added [IETF RFC-7807](https://tools.ietf.org/html/rfc7807) compatible problem
  responses on errors instead of just strings. This is a breaking change if
  you're depending on the error message syntax from before. (!45, !46)

- Added Go module caching in the Dockerfile so local iterations will be
  slightly quicker to compile. (!44)

- Added endpoint `GET /version` that returns an object of version data of the
  API itself. (#2)

- Added Swagger spec metadata such as version that equals the version of the
  API, contact information, and license. (#2)

- Changed version of GORM from v1 to v2. Some constraints have been renamed in
  the database, and migrations were added to automatically upgrade the database.
  (!40)

- Changed version of github.com/gin-gonic/gin from v1.4.0 to v1.7.1. (!45)

- Changed GORM `.Where()` clause usage throughout the repo to use less
  positional references and more named references of fields. (!41)

- Changed Go runtime from v1.13 to v1.16. (!44)

- Changed to use `c.ShouldBindJSON` instead of `c.BindJSON` throughout the repo.
  This should result in fewer warnings being logged by gin-gonic. See issue
  [iver-wharf/iver-wharf.github.io#22](https://github.com/iver-wharf/iver-wharf.github.io/issues/22)
  for more information. (!46)

- Changed unimplemented endpoints to return HTTP 501 (Not Implemented)
  status code. Including: (!46)

  - `GET /branch/{branchid}`
  - `GET /branches`
  - `POST /builds/search`

- Removed DB column `token.provider_id` as it was causing a circular reference
  between the `token` and `provider` tables. The `POST /token` endpoint still
  accepts a `providerId` field, but this may be removed in the future. (!40)

- Removed gin-gonic logging endpoints `GET /logs` and `POST /logs`. (!47)

## v3.0.0 (2021-04-13)

- Added ability to sort builds in the `GET /projects/{projectid}/builds`
  endpoint using the new `orderby` query parameter, which defaults to
  `?orderby=buildId desc`. (!35)

- Added way to disable RabbitMQ integration via `RABBITMQENABLED` environment
  variable. If it's unset, empty, or `false` then the integration is disabled.
  If it's `true` then it's enabled. This means it will be disabled by default.
  (!38)

- Changed to use new open sourced `wharf-message-bus-sender` package
  [github.com/iver-wharf/messagebus-go](https://github.com/iver-wharf/messagebus-go)
  and bumped said package version from v0.1.0 to v0.1.1. (!34)

- Changed time fields to be marked with the OpenAPI format `date-time` in the
  build and log models so they can be appropriately parsed as `Date` in the
  TypeScript code generated by swagger-codegen-cli. (!36)

- Removed unused `random.go`. (!34)

- Added `GIT_BRANCH` to jobs params. (!37)

## v2.1.0 (2021-03-15)

- Added CHANGELOG.md to repository. (!26)

- Added `PUT /build/{buildid}?status={status}` to update build status. (!27)

- Changed build status naming. Moved to new file. (!28)

- Updated swag to version 1.7.0. (!30)

- Fixed swag comment to allow posting an artifact file. (!30)

- Fixed issue with wrong arifact downloaded from
  `/build/{buildid}/artifact/{artifactId}` endpoint. (!30)

- Updated swagger type from external library null.String to string. (!32)

## v2.0.0 (2021-01-15)

- Removed Wharf group database table migration that was introduced in v1.0.0.
  (!25)

- Removed `tfs` -> `azuredevops` provider name migration that was introduced
  in v0.8.0. (!25)

- Removed name column datatype change in branch table (to `varchar`)
  migration that was introduced in v1.0.1. (!25)

## v1.1.0 (2021-01-14)

- Changed `POST /project/{projectid}/{stage}/run` so the `branch` parameter is
  now optional, where it will use the default branch name if omitted. (!23)

- Changed `POST /project/{projectid}/{stage}/run` so the `environment` parameter
  is now optional, where it will start the build with no environment filter
  set. (!23)

- Changed the "environment" column in the "build" table to be nullable.
  Automatic migrations will be applied. (!23)

- Fixed `GET /stream` endpoint to sanitize JSON of data sent to `POST /log`
  endpoint. (!24)

## v1.0.1 (2021-01-07)

- Removed `project.BuildHistory` from all `GET /project/*` and
  `GET /projects/*` endpoints. (!18)

- Added new endpoint `GET /projects/{projectid}/builds` to get a paginated
  result of the list of builds. (!18)

- Added dependency on `github.com/sirupsen/logrus`. (!22)

- Fixed minor logging inconsistencies. (!22)

- Fixed wrong parameter type for `artifactId` in OpenAPI definition for
  `GET /build/{buildid}/artifact/{artifactId}`, `string` -> `int`. (!22)

- Changed name column datatype in branch table from `varchar(100)` to `varchar`,
  effectively removing the limit on the length of a project's name (!21).

## v1.0.0 (2020-11-27)

- Added implementation of `PUT /branches` endpoint. (!16)

- Added migrations to move the group data from the group database table to be
  embedded in the project table. (!20)

- Removed the group database table. (!20)

## v0.8.0 (2020-10-23)

- Changed `tfs` provider name to `azuredevops`, with migrations to update the
  database. (!17)

## v0.7.10 (2020-10-20)

- Added dependency on `wharf-message-bus-sender`. (!15)
- Added publishing of build events via RabbitMQ. (!15)

## v0.7.9 (2020-07-03)

- Added sending of `WHARF_INSTANCE` as a query parameter to the Jenkins webhook
  trigger. Value is taken from the `WHARF_INSTANCE` environment variable. (!14)

- Added sending of `REPO_NUMBER` as a query parameter to the Jenkins webhook
  trigger. Value is the project ID, taken from the database. (f9d794c)

## v0.7.8 (2020-06-23)

- Changed list of branches to be sorted by name in all projects endpoints. (!13)

## v0.7.7 (2020-06-04)

- Changed Dockerfile to use Iver self-signed cert from inside the repo at
  `/dev/ca-certificates.crt`. (!12)

## v0.7.6 (2020-06-03)

- Fixed configured cert path in Dockerfile. (!11)

## v0.7.5 (2020-05-29)

- Added Iver self-signed certs to be loaded into the Go `http.DefaultClient`.
  (!10)

## v0.7.2 (2020-05-06)

- Added missing OpenAPI parameters to the
  `POST /project/{projectid}/{stage}/run` endpoint. (!9)

- Added dummy reponse to Jenkins response to not need Jenkins locally, and the
  accompanying added `MOCK_LOCAL_CI_RESPONSE` environment variable to enable it.
  (!8)

- Added new fields to the "build" database table to be populated when starting
  a new build: "gitBranch", "environment", and "stage". (!7)

## v0.7.1 (2020-03-10)

- Changed artifact retreival to refer to the artifact ID instead of the
  artifacts name. `GET /build/{buildid}/artifact/{artifactname}` ->
  `GET /build/{buildid}/artifact/{artifactid}`. (!5)

- Changed OpenAPI return type of `GET /build/{buildid}/artifact/{artifactid}`
  to `file` (BLOB). (!6)

- Fixed logs getting spammed by health checks. Health checks are no longer
  visible in the logs. (!4)

- Fixed `DBLOG` setting only checking if the environment variable was present.
  It now parses the `DBLOG` environment variable as a boolean. (!4)

- Fixed branches not getting loaded in the `GET /projects` endpoint as they
  are in the `GET /project/{projectid}` endpoint. (!3)

## v0.7.0 (2020-02-20)

- Added test results endpoint `GET /tests-results` to return test result summary
  based on uploaded `.trx` artifacts. (!2)

## v0.6.0 (2020-02-04)

- Changed `JENKINS_URL` and `JENKINS_TOKEN` environment variables to `CI_URL`
  and `CI_TOKEN`, respectively. (b0ed707)

- Added "token" foreign key association from the "project" table to the "token"
  table in the database. (b0ed707)

- Added passing of Git provider access token to Jenkins via `GIT_TOKEN` query
  parameter. (b0ed707)

- Removed "job" table from the database. (0d85ddf, af35d71, f9bfbdd)

- Changed `swag init` build in Dockerfile to use `go.mod`. (e284dde)

## v0.5.5 (2020-01-17)

- Changed swaggo build to use Gomodules instead of fixed set of dependencies.
  Cleaning up the Dockerfile. (30110ed)

- Added search endpoint for projects `POST /projects/search`. (!1)

- Added `.wharf-ci.yml` to build via Wharf.

- Removed Wharf API Go client package (`/pkg/client/`) out into its own
  repository. (9685945)

- Added this repository by splitting it out from the joined repo. (a7d5f00)
