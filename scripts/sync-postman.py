import json
import os
import sys
import urllib.error
import urllib.request
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

request = urllib.request.Request(
    f"https://api.postman.com/collections/{collection_id}",
    data=json.dumps({"collection": payload}, separators=(",", ":")).encode("utf-8"),
    headers={"X-API-Key": api_key, "Content-Type": "application/json"},
    method="PUT",
)
try:
    with urllib.request.urlopen(request, timeout=30) as response:
        result = json.load(response)
except urllib.error.HTTPError as error:
    detail = error.read().decode("utf-8", errors="replace")
    sys.exit(f"Postman collection sync failed with HTTP {error.code}: {detail}")

synced = result.get("collection", {})
print(f"Synced Postman collection: {synced.get('name', payload['info']['name'])} ({synced.get('id', collection_id)})")

