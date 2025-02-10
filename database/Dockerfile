# Use the official PostgreSQL image from the Docker Hub
FROM postgres:latest

# Set environment variables for PostgreSQL
ENV POSTGRES_USER=admin
ENV POSTGRES_PASSWORD=adminpassword
ENV POSTGRES_DB=battleship

# Copy the initialization SQL script into the container
COPY ./init.sql /docker-entrypoint-initdb.d/


# Expose the default PostgreSQL port
EXPOSE 5432


# The default command to run PostgreSQL
CMD ["postgres"]