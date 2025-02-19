---
# Technology Stack

This document outlines the technology stack used in the Pavilion Network backend.

## Programming Language

- **Go (Golang)**: The primary programming language used for backend development. It provides excellent performance, concurrency support, and a robust standard library.

## Web Framework

- **Gin**: A lightweight and fast HTTP web framework for Go, used for building RESTful APIs.

## ORM & Database

- **GORM**: An ORM library for Go, used to interact with the database.
- **CockroachDB**: A scalable, distributed SQL database used as the primary data store. CockroachDB is compatible with the PostgreSQL driver but offers high availability and horizontal scalability ideal for large-scale systems.

## Caching

- **Redis**: Used as an in-memory data store and cache to improve performance and scalability.

## Authentication & Authorization

- **JWT**: JSON Web Tokens are used for secure session management and API authentication.
- **OAuth**: The backend supports OAuth integrations (e.g., Google, Apple, Ethereum) for third-party authentication.

## Storage Services

- **IPFS**: The InterPlanetary File System is used for decentralized file storage.
- **S3 (Amazon S3/MinIO)**: Used for object storage, particularly for media files.

## API Documentation

- **Swagger (Swaggo)**: Integrated with Gin to generate and serve API documentation for the backend.

## Development & Deployment

- **Docker**: Containerization is optionally used to ensure consistent deployment environments.
- **CI/CD**: Integration with continuous integration and deployment pipelines for automated testing and streamlined deployments.

## Monitoring & Logging

- **Zap Logger**: Logging is implemented using Uber's Zap library for high-performance structured logging.
- Custom monitoring tools and metrics are integrated for comprehensive application performance and reliability tracking.

--- 