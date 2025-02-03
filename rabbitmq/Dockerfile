# Use the official RabbitMQ image with management plugin from Docker Hub
FROM rabbitmq:management

# Expose RabbitMQ ports (default and management UI)
EXPOSE 5672 15672

# You can set environment variables for RabbitMQ as needed
ENV RABBITMQ_DEFAULT_USER=admin
ENV RABBITMQ_DEFAULT_PASS=adminpassword

# The default command to run RabbitMQ
CMD ["rabbitmq-server"]
