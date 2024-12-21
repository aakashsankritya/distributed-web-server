## Distributed web-server
The distributed web server is designed to:

- deduplicating unique request based on id 
- calling endpoints if provided
- event logging to kafka

## Tech Stack and components
- Webserver: Golang for its concurrency support and performance.
- Redis: Used as a deduplication store for request IDs.
- Kafka & Zookeeper: For event logging and distributed message streaming.
- Nginx: To balance load across N web server instances (currently, the system uses 3 instances in the Docker setup).

## Thought process

I came accross multiple approaches for the above requirements:

# Approach 1: In-Memory Deduplication with Coordinator

- Each webserver maintains their own set of unique request they served in-memory Map
- One server acts as a coordinator:
    - It collects the unique request sets from all servers (calling an API)
    - Aggregates the data.
    - Logs events to a file or Kafka.

* Advantages:
- Simple design with no dependency on external services.
* Disadvantages:
- Time-Consuming: Aggregation across servers adds latency.
- Non-Durable: If a server crashes, the in-memory data is lost, leading to event loss.

# Approach 2: Enhanced In-Memory with Write-Ahead Logging (WAL) 

- Approach 2 is a bit of enhancement to Approach 1, where each server can maintain WAL (Write-ahead-logging) to persist unique request IDs.
- Every minute, the coordinator:
    - Collects WAL data from all servers.
    - Aggregates the logs.
    - Logs events to a file or Kafka.

* Advantages:
- Adds durability through WAL, reducing data loss risks.

* Disadvantages:
- Slightly more complex than Approach 1.
- Coordination overhead persists, potentially impacting performance.

# Final Approach: Redis-Centric Deduplication

- Global Deduplication:
    - Use Redis as a centralized store for deduplication (SET<id1, id2, ... idN>).
    - Redis ensures high performance and durability for storing request IDs.
- Only the server that successfully acquires the lock:
    - Retrieves the count of unique request IDs.
    - Logs the aggregated data to a file or Kafka.
    - Flushes the Redis SET.
- Concurrency Handling:
    - Multiple servers run independently without direct coordination.
    - The Redis lock ensures only one server performs aggregation at a time.

* Advantages:
- Durable: Redis persists data, minimizing event loss.
- Efficient: Eliminates inter-server communication for deduplication.
- Scalable: Suitable for high-load scenarios with >10K requests/sec.

Disadvantages:
- Requires Redis as an external dependency.

# Webserver Design
* High Performance
- Designed to handle 10K requests/sec using a worker pool model:
- The worker pool is configurable based on available resources (CPU, memory).
- Workers process requests concurrently, ensuring optimal throughput.

* Load Balancing
- Nginx distributes incoming requests across N web server instances.
- This ensures even load distribution and high availability.

* Scalability
- The modular design enables scaling by adding more web servers or increasing Redis capacity.
