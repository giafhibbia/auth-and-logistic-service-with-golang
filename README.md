

# **Microservices Project**

This repository contains microservices for authentication and logistics, built with **Go** and using **MongoDB**, **PostgreSQL**, and **RabbitMQ**. All services run in Docker containers for easy local development.

* * *

**Prerequisites**

*   **Go** v1.22.4 (Windows amd64)  
    Check with:

bash

CopyEdit

go version

*   **Docker & Docker Compose** (tested with Docker 28.2.2)  
    Check with:

bash

CopyEdit

docker --version

docker compose version

* * *

**Services Overview**

*   **MongoDB (v7)**: NoSQL database for authentication and logistics data.
*   **PostgreSQL (v16)**: Relational database for worker services.
*   **RabbitMQ (v3-management)**: Message broker for event-driven communication.
*   **Auth Service**: User registration, login, and JWT authentication (Go).
*   **Logistic Service**: Shipment creation, status update, and tracking (Go).
*   **Worker**: Background message processing from RabbitMQ (Go).
*   **Swagger UI**: API documentation interface.

* * *

**Project Structure**

bash

CopyEdit

/

├── auth-service         # Auth microservice (Go)

├── logistic-service     # Logistic microservice (Go)

├── worker               # Background worker (Go)

├── api-docs             # OpenAPI specs (swagger.yaml)

├── docker-compose.yaml

└── README.md

* * *

**How to Run Locally**

1.  **Clone this repository:**

bash

CopyEdit

git clone https://github.com/giafhibbia/logistics-auth-service.git

cd logistics-auth-service

2.  **Start all services:**

bash

CopyEdit

docker compose up --build

3.  **Verify services:**

*   **MongoDB**: localhost:27017
*   **PostgreSQL**: localhost:5432
*   **RabbitMQ Management UI**: [http://localhost:15672](http://localhost:15672) (guest/guest)
*   **Auth Service API**: [http://localhost:8081](http://localhost:8081)
*   **Logistic Service API**: [http://localhost:8082](http://localhost:8082)
*   **Swagger UI**: [http://localhost:8083](http://localhost:8083) (loads API docs from ./api-docs/swagger.yaml)

* * *

**Environment Variables**

**Auth Service & Logistic Service**

*   MONGO\_URI — MongoDB connection string (e.g., mongodb://mongo:27017)
*   JWT\_SECRET — JWT signing key
*   RABBITMQ\_URL — RabbitMQ connection string (e.g., amqp://guest:guest@rabbitmq:5672/)

**Worker**

*   RABBITMQ\_URL — RabbitMQ connection string
*   MASTERDB\_URL — PostgreSQL connection string (e.g., postgres://master:password@postgres:5432/masterdb?sslmode=disable)

* * *

**Docker Compose Services**

| Service | Description | Port(s) |
| --- | --- | --- |
| mongo | MongoDB NoSQL database | 27017 |
| postgres | PostgreSQL relational database | 5432 |
| rabbitmq | RabbitMQ broker & UI | 5672, 15672 |
| auth-service | Auth microservice (Go) | 8081 |
| logistic-service | Logistics microservice (Go) | 8082 |
| worker | Background async worker (Go) | - |
| swagger-ui | API documentation interface | 8083 |

* * *

**Useful Commands**

*   **Start all containers:**

bash

CopyEdit

docker compose up --build

*   **Stop and remove all containers:**

bash

CopyEdit

docker compose down

*   **View logs for a service:**

bash

CopyEdit

docker logs -f auth-service

* * *

**Troubleshooting**

*   If ports are already in use, update the ports in docker-compose.yaml.
*   If services fail to connect, check Docker networking and environment variables.
*   Use the RabbitMQ Management UI ([http://localhost:15672](http://localhost:15672)) to monitor queues and messages.

* * *

**API Documentation**

The OpenAPI/Swagger documentation is available at [http://localhost:8083](http://localhost:8083).

To update the API docs, edit the OpenAPI spec at /api-docs/swagger.yaml.