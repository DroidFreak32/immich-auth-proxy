 # Immich Auth Proxy

 A simple reverse proxy to bridge a self-hosted [Immich](https://immich.app/) server with its `immich-machine-learning` service running on a secure Google Cloud environment (like Cloud Run).

 The main Immich server does not natively support adding the OIDC identity tokens required to authenticate with a secured Cloud Run service. This proxy solves that problem by alongside your self-hosted Immich instance and intercepting requests from Immich to the machine-learning service, attaching the necessary `Authorization: Bearer <token>` header, and forwarding them.

 ## Prerequisites

 *   An `immich-machine-learning` service deployed on Google Cloud (e.g., on Cloud Run) that requires authentication.
 *   A Google Cloud Service Account with permissions to invoke the service and create ID tokens.
 *   An Immich server instance, of course.

 ## Configuration

 The proxy is configured via environment variables:

 | Variable                         | Description                                                                                                                            | Required | Example                                       |
 | -------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- | -------- | --------------------------------------------- |
 | `UPSTREAM_SERVER_URL`            | The full URL of the upstream `immich-machine-learning` service.                                                                        | **Yes**  | `https://my-immich-ml-xyz.a.run.app`          |
 | `PORT`                           | The port on which the proxy server will listen. Defaults to `8080`.                                                                    | No       | `8080`                                        |
 | `GOOGLE_APPLICATION_CREDENTIALS` | For local development, set this to the path of your downloaded service account JSON key file. Not needed when deployed on Google Cloud. | No       | `/path/to/your/sa-key.json`                   |

 ## Usage

 ### Local Development

 1.  Download a JSON key for your service account.
 2.  Set the environment variables:
     ```sh
     export GOOGLE_APPLICATION_CREDENTIALS="/path/to/sa-key.json"
     export UPSTREAM_SERVER_URL="https://your-immich-ml-service.a.run.app"
     ```
 3.  Run your chosen implementation from its directory:
     *   **Go:**
         ```sh
         cd golang
         go run main.go
         ```
     *   **Python:**
         ```sh
         cd python
         pip install -r requirements.txt
         gunicorn --bind 127.0.0.1:8080 proxy:app
         ```
Now set the ML URL to `http://localhost:8080`

 ### Docker

 Both implementations include a `Dockerfile`.

 > **Note on Security:** The provided `Dockerfile`s may copy a local `sa.json` file into the container image. This is not recommended for production. The following examples show the best practice of mounting the service account key as a volume at runtime.

 ```sh
 # 1. Build the image
 docker build -t immich-auth-proxy:latest ./golang # or ./python

 # 2. Run the container, mounting the service account key
 docker run -p 8080:8080 \
   -e UPSTREAM_SERVER_URL="https://your-immich-ml-service.a.run.app" \
   -e GOOGLE_APPLICATION_CREDENTIALS="/app/sa.json" \
   -v /path/to/your/sa-key.json:/app/sa.json:ro \
   immich-auth-proxy:latest
 ```

 #### Docker compose example:

```yaml
...

  ml-proxy:
    build:
      context: /PATH/TO/THIS/REPOSITORY/golang
      dockerfile: Dockerfile
      # Avoid strict firewall rules blocking access to internet during build
      network: host
    container_name: ml-proxy
    image: ml-proxy:latest
    env_file:
      - .env # Contains the UPSTREAM_SERVER_URL
    volume:
      - /path/to/your/sa-key.json:/app/sa.json:ro
    restart: always
...
```
Now set the ML URL to `http://ml-proxy:8080`