# Wharf API

[![Codacy Badge](https://app.codacy.com/project/badge/Grade/157d2eff9eba41a4a5deee8bb748a9f5)](https://www.codacy.com/gh/iver-wharf/wharf-api/dashboard?utm_source=github.com\&utm_medium=referral\&utm_content=iver-wharf/wharf-api\&utm_campaign=Badge_Grade)

The API is one of the main components in Wharf. Its purpose is mainly a
[CRUD](https://en.wikipedia.org/wiki/Create,\_read,\_update_and_delete)
interface on top of the database that the other components in Wharf interact
with.

## Components

- HTTP API using the [gin-gonic/gin](https://github.com/gin-gonic/gin)
  web framework.

- Swagger documentation generated using
  [swaggo/swag](https://github.com/swaggo/swag) and hosted using
  [swaggo/gin-swagger](https://github.com/swaggo/gin-swagger).

- Database [ORM](https://en.wikipedia.org/wiki/Object%E2%80%93relational_mapping)
  using [gorm.io/gorm](https://gorm.io/).

## Configuring

The wharf-api program can be configured via environment variables and through
optional config files. See the docs on the `Config` type over at:
<https://pkg.go.dev/github.com/iver-wharf/wharf-api#Config>

## Development

1. Install Go 1.16 or later: <https://golang.org/>

2. Install the [swaggo/swag](https://github.com/swaggo/swag) CLI globally:

   ```sh
   # Run this outside of any Go module, including this repository, to not
   # have `go get` update the go.mod file.
   $ cd ..

   $ go get -u github.com/swaggo/swag
   ```

3. Generate the swaggo files (this has to be redone each time the swaggo
   documentation comments has been altered):

   ```sh
   # Navigate back to this repository
   $ cd wharf-api

   # Generate the files into docs/
   $ swag init --parseDependency --parseDepth 1
   ```

4. Start hacking with your favorite tool. For example VS Code, GoLand,
   Vim, Emacs, or whatnot.

## Linting Golang

- Requires Node.js (npm) to be installed: <https://nodejs.org/en/download/>
- Requires Revive to be installed: <https://revive.run/>

```sh
go get -u github.com/mgechev/revive
```

```sh
npm run lint-go
```

## Linting markdown

- Requires Node.js (npm) to be installed: <https://nodejs.org/en/download/>

```sh
npm install

npm run lint-md

# Some errors can be fixed automatically. Keep in mind that this updates the
# files in place.
npm run lint-md-fix
```

## Linting

You can lint all of the above at the same time by running:

```sh
npm run lint

# Some errors can be fixed automatically. Keep in mind that this updates the
# files in place.
npm run lint-fix
```

---

Maintained by [Iver](https://www.iver.com/en).
Licensed under the [MIT license](./LICENSE).
