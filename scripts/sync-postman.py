import json
import os
import subprocess
import sys
from pathlib import Path


api_key = os.environ.get("POSTMAN_API_KEY", "")
collection_id = os.environ.get("POSTMAN_COLLECTION_ID", "")
if not api_key and not collection_id:
    print("Postman sync skipped: set POSTMAN_API_KEY and POSTMAN_COLLECTION_ID in repository secrets.")
    sys.exit(0)
if not api_key or not collection_id:
    sys.exit("Both POSTMAN_API_KEY and POSTMAN_COLLECTION_ID must be set.")

collection_path = Path("postman/ci/performance-engine.postman_collection.json")
with collection_path.open(encoding="utf-8") as collection_file:
    payload = json.load(collection_file)

body = json.dumps({"collection": payload}, separators=(",", ":"))
command = [
    "curl",
    "--silent",
    "--show-error",
    "--fail-with-body",
    "--max-time",
    "30",
    "--request",
    "PUT",
    f"https://api.postman.com/collections/{collection_id}",
    "--header",
    f"X-API-Key: {api_key}",
    "--header",
    "Content-Type: application/json",
    "--header",
    "Accept: application/json",
    "--data-binary",
    "@-",
    "--write-out",
    "\n%{http_code}",
]
response = subprocess.run(command, input=body, text=True, capture_output=True, check=False)
response_body, _, status_code = response.stdout.rpartition("\n")
if response.returncode != 0 or not status_code.startswith("2"):
    detail = response_body or response.stderr.strip() or "No response body"
    if "1010" in detail:
        sys.exit(
            "Postman collection sync was blocked by its Cloudflare security layer "
            f"(HTTP {status_code or 'unknown'}, error 1010). The request used curl; "
            "contact Postman Support with the Cloudflare Ray ID from the response."
        )
    sys.exit(f"Postman collection sync failed with HTTP {status_code or 'unknown'}: {detail}")

result = json.loads(response_body)
synced = result.get("collection", {})
print(f"Synced Postman collection: {synced.get('name', payload['info']['name'])} ({synced.get('id', collection_id)})")

