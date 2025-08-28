# Use an official Python runtime as a parent image
FROM python:3.11-slim

# Set environment variables
ENV PYTHONUNBUFFERED=True
ENV APP_HOME=/app

# Create and set the working directory
WORKDIR $APP_HOME

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the application code into the container
COPY proxy.py .
COPY sa.json .
# Define the command to run the application using gunicorn
# Gunicorn is a production-ready WSGI server
# It will listen on the port specified by the $PORT environment variable,
# which is automatically set by services like Cloud Run.
CMD exec gunicorn --bind 0.0.0.0:3003 --workers 1 --threads 8 --timeout 0 proxy:app
#CMD ["exec", "gunicorn", "--bind", "0.0.0.0:$PORT", "--workers", "1", "--threads", "8", "--timeout", "0", "proxy:app"]
