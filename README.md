# Owlet Sock Scraper

This application polls the API service (Ayla) which serves as a backend for the Owlet Smart Sock. 

## Why?
The app kinda sucks

## How?
This is based on the work of others (e.g. this [Python implementation](https://github.com/BastianPoe/owlet_api)), but has been tweaked and edit to suit my needs. It's also written in Go.

### Running Locally
To run locally, you will need a Postgres database running on port 5432. You will also need to populate a `.env` file with the following:
```env
# Same as the Owlet mobile app
OWLET_USERNAME='...'
OWLET_PASSWORD='...'
OWLET_REGION="europe"

# Postgres
POSTGRES_USER=postgres
POSTGRES_PASSWORD=password
POSTGRES_HOST=localhost
POSTGRES_PORT="5432"
POSTGRES_DB=owletDB

# Grafana
GF_SECURITY_ADMIN_PASSWORD=admin
```

#### Development
To run while developing, run the following from the repo root:
```shell
go run .
```

#### Dockerised
Or build the docker image and the whole setup via Docker Compose:
```shell
docker build -t edfincham/owlet-sock-scraper:latest .
docker compose up
```

### Grafana
A dashboard JSON for display the heart rate and blood oxygenation is provided within the `grafana` directory. To get this to work, a Postgres data source must be configured in the Grafana UI. Assuming the example `.env` is used, the following fields should work:

#### Connection
 - Name: `grafana-postgresql-datasource`
 - Host URL:
    - Docker Compose: `db:5432`
    - Kubernetes: `postgres.owlet` (i.e. `<service_name>.<namespace>`)
 - Database name: `owletDB`

#### Authentication
 - Username: `postgres`
 - Password: ...
 - TLS/SSL Mode: `disable`

#### Additional Settings
 - PostgreSQL Version: `15`
