import os
import requests
from flask import Flask, request, Response

# Imports for Google Cloud authentication
import google.auth
from google.auth.transport.requests import Request as GoogleAuthRequest
from google.oauth2 import id_token

app = Flask(__name__)

# Fetch the target server URL from environment variables.
# This is the server your proxy will forward requests to.
UPSTREAM_SERVER_URL = os.environ.get("UPSTREAM_SERVER_URL")
if not UPSTREAM_SERVER_URL:
    raise ValueError("UPSTREAM_SERVER_URL environment variable not set.")

# --- Google Cloud ID Token Generation ---
def get_google_id_token(audience):
    """
    Generates a Google-signed OIDC ID token for the specified audience.
    
    This function uses the Application Default Credentials (ADC) strategy.
    It will automatically use the GOOGLE_APPLICATION_CREDENTIALS environment
    variable for local testing or the attached service account when deployed
    on Google Cloud (e.g., Cloud Run, GKE).
    """
    try:
        # Create a request object for the authentication library.
        auth_request = GoogleAuthRequest()
        
        # Fetch the ID token. The audience is the URL of the service being called.
        token = id_token.fetch_id_token(auth_request, audience)
        
        return token
    except Exception as e:
        print(f"Error generating ID token: {e}")
        return None

# --- Main Proxy Route ---
@app.route('/', defaults={'path': ''}, methods=['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS'])
@app.route('/<path:path>', methods=['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS'])
def proxy_request(path):
    """
    Catches all incoming requests, adds the auth header, and forwards them.
    """
    print(f"Received request for path: {path}")

    # 1. Generate the Google ID Token
    # The audience for the token should be the root URL of the target server.
    token = get_google_id_token(audience=UPSTREAM_SERVER_URL)
    
    if not token:
        return Response("Could not generate authentication token.", status=500, mimetype='text/plain')

    # 2. Prepare the request for forwarding
    destination_url = f"{UPSTREAM_SERVER_URL}/{path}"
    
    # Copy headers from the original request
    forward_headers = {key: value for key, value in request.headers}

    # Add the Bearer token to the Authorization header
    forward_headers["Authorization"] = f"Bearer {token}"
    
    # Host header should be the destination's host
    forward_headers["Host"] = UPSTREAM_SERVER_URL.split('//')[1].split('/')[0]
    
    print(f"Received headers: {forward_headers}")

    # 3. Forward the request using the 'requests' library
    try:
        resp = requests.request(
            method=request.method,
            url=destination_url,
            headers=forward_headers,
            data=request.get_data(),
            params=request.args,
            stream=True,  # Use stream to handle large responses efficiently
            timeout=300    # Set a reasonable timeout
        )
    except requests.exceptions.RequestException as e:
        print(f"Error forwarding request: {e}")
        return Response(f"Proxy request failed: {e}", status=502)

    # 4. Return the response from the upstream server to the original client
    # Exclude headers that are managed by the proxy's server (e.g., gunicorn)
    excluded_headers = ['content-encoding', 'content-length', 'transfer-encoding', 'connection']
    response_headers = [
        (key, value) for key, value in resp.raw.headers.items()
        if key.lower() not in excluded_headers
    ]

    return Response(resp.content, resp.status_code, response_headers)

if __name__ == '__main__':
    # Use 0.0.0.0 to make it accessible outside the container
    # The port is fetched from the PORT env var, default to 8080
    port = int(os.environ.get('PORT', 8080))
    app.run(host='0.0.0.0', port=port)

