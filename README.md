# Auth Service

A scalable Authentication Service written in Go.

## Table of Contents

- [Project Structure](#project-structure)
- [Makefile](#makefile)
- [Running the Application](#running-the-application)
  - [Using Docker](#using-docker)
  - [Without Docker](#without-docker)
- [API Documentation](#api-documentation)
- [Database Migrations](#database-migrations)
- [Application Configuration](#application-configuration)
- [License](#license)

## Project Structure

Here are the main components of the application:

```bash
.
├── api                   # API documentation
├── cmd                   # Application entrypoint
├── internal
│   ├── config            # Configuration loading logic
│   ├── entity            # Core domain etities
│   ├── http              # Data delivery level
│   │   └── handler
│   │       └── v1
│   ├── repository        # Database repositories
│   │   └── postgres
│   └── service           # Business logic
├── migrations
└── pkg
    ├── notifier          # User notification logic
    └── postgres          # PostgreSQL connection and migration logic

```

## Makefile

Explore avaliable `Makefile` targets:

```bash
make help
```

## Running the Application

### Using Docker

1. Prepare config file.

2. Run the application:

    ```bash
    # You can ovveride CONFIG_PATH (Default: ./config.yml)
    make docker-compose-up CONFIG_PATH=<path>
    ```

### Without Docker

1. Setup PostgreSQL.

2. Prepare config file.

3. Run the application:

    ```bash
    # You can ovveride CONFIG_PATH (Default: ./config.yml)
    make all CONFIG_PATH=<path>
    ```

## API Documentation

The application is documented using Swagger. You can explore the API in `api/swagger.yml`.

## Database Migrations

The application automatically applies migrations from the `/migrations` directory, but you can run them manually using the `Makefile`:

```bash
# Create migration
MIGRATION_NAME=<name> make migrate-create

# Apply all migrations
DATABASE_DSN=<dsn> make migrate-up

# Rollback all migrations
DATABASE_DSN=<dsn> make migrate-down
```

## Application Configuration

The application is configured via YAML file. Application uses `-configPath` flag to load configuration from YAML file. Default value is `./config.yml`.

```bash
# You can override default value
CONFIG_PATH=<path> make run
```

Here is the basic structure of the configuration file:

```yaml
env: dev

tokens:
  # required
  access_secret: QZ8PVsIRj5SbSvjpMUp75fUzWydyQY4zXBvtoir532E=
  access_ttl: 15m
  # required
  refresh_secret: +1bKAkSTo3Iluk3g8pU0Fe45kLJf1zA6shjDbAAhk1I=
  refresh_ttl: 1h

server:
  port: 8080
  read_timeout: 5s
  write_timeout: 10s
  idle_timeout: 1m
  max_header_bytes: 1048576 # 1 << 20

postgres:
  # required
  user: postgres
  # required
  password: postgres
  host: db
  port: 5432
  # required
  database: auth-service
  sslmode: disable
  max_conns: 20
  min_conns: 4
  max_conn_lifetime: 1h
  max_conn_idle_time: 30m
  health_check_period: 1m
  migrations_path: ./migrations
```

The behavior of the application depends on the `env` passed in the configuration file:

1. `dev` - logging is structured with plain text (debug level).
2. `stage` - logging is structured with JSON (debug level).
3. `prod` - logging is structured with JSON (info level).

## License

This project is licensed under the WTFPL License - see the `LICENSE` file for details.
