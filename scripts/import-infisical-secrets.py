#!/usr/bin/env python3
"""Temporary GitHub-to-Infisical import from a dedicated branch."""
import json
import os
from pathlib import Path
import re
import sys
import time
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import HTTPRedirectHandler, Request, build_opener

PROJECT_ID = "1811949a-baed-427c-9eb1-9525f6b43964"
API_URL = "https://app.infisical.com/api/v4/secrets"


class MigrationError(Exception):
    pass


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise MigrationError("Infisical redirected the request; verify the configured host.")


class InfisicalClient:
    def __init__(self, token, environment, path):
        if not token:
            raise MigrationError("INFISICAL_MIGRATION_TOKEN is missing.")
        self.token = token
        self.location = {
            "projectId": PROJECT_ID,
            "environment": environment,
            "secretPath": path,
            "type": "shared",
        }
        self.opener = build_opener(NoRedirect())

    def request(self, method, name, *, query=None, body=None, allow_missing=False):
        url = f"{API_URL}/{name}"
        if query:
            url += "?" + urlencode(query)
        data = None if body is None else json.dumps(body).encode("utf-8")
        request = Request(url, data=data, method=method, headers={
            "Authorization": f"Bearer {self.token}",
            "Content-Type": "application/json",
        })
        for attempt in range(4):
            try:
                with self.opener.open(request, timeout=30) as response:
                    return json.load(response)
            except HTTPError as error:
                if allow_missing and error.code == 404:
                    error.close()
                    return None
                if error.code != 429 or attempt == 3:
                    error.close()
                    raise MigrationError(f"Infisical request failed (HTTP {error.code}).") from None
                retry_after = (error.headers or {}).get("Retry-After")
                error.close()
                if retry_after is None:
                    delay = min(60 * 2 ** attempt, 120)
                elif re.fullmatch(r"[0-9]{1,3}", retry_after):
                    delay = max(1, int(retry_after))
                else:
                    raise MigrationError("Infisical rate-limit delay requires a later manual retry.") from None
                if delay > 120:
                    raise MigrationError("Infisical rate-limit delay exceeds the retry budget; retry later.") from None
                print(f"Infisical rate limit reached; retrying in {delay} seconds.", file=sys.stderr)
                time.sleep(delay)
            except (URLError, TimeoutError, OSError, ValueError):
                raise MigrationError("Infisical request failed; response details are suppressed.") from None

    def read(self, name):
        response = self.request("GET", name, query={
            **self.location,
            "viewSecretValue": "true",
            "expandSecretReferences": "false",
            "includeImports": "false",
        }, allow_missing=True)
        if response is None:
            return None
        secret = response.get("secret", {})
        if (secret.get("secretKey") != name
                or secret.get("secretValueHidden")
                or not isinstance(secret.get("secretValue"), str)):
            raise MigrationError(f"Cannot read the shared value for {name}.")
        return secret["secretValue"]

    def create(self, name, value):
        self.request("POST", name, body={
            **self.location,
            "secretValue": value,
            "skipMultilineEncoding": True,
        })


def source_values(names, environment):
    if (not names or len(names) != len(set(names))
            or any(not re.fullmatch(r"[A-Z_][A-Z0-9_]*", name) for name in names)):
        raise MigrationError("The secret-name inventory is invalid.")
    values = {name: environment.get(f"SOURCE_{name}", "") for name in names}
    missing = [name for name, value in values.items() if value == ""]
    if missing:
        raise MigrationError("Missing GitHub values: " + ", ".join(missing))
    return values


def migrate(client, source, mode):
    if mode not in ("dry-run", "apply", "verify"):
        raise MigrationError("Unknown migration mode.")
    # Infisical may expand these later when syncing back to GitHub.
    references = [name for name, value in source.items() if "${" in value]
    if references:
        raise MigrationError("Literal reference syntax needs manual review: " + ", ".join(references))

    existing = {name: client.read(name) for name in source}
    conflicts = [name for name, value in existing.items() if value is not None and value != source[name]]
    if conflicts:
        raise MigrationError("Existing Infisical values differ; nothing written: " + ", ".join(conflicts))
    missing = [name for name, value in existing.items() if value is None]
    if mode == "verify" and missing:
        raise MigrationError("Missing Infisical secrets: " + ", ".join(missing))

    for name, value in source.items():
        if name not in missing:
            print(f"MATCH {name}")
        elif mode == "dry-run":
            print(f"WOULD_CREATE {name}")
        else:
            client.create(name, value)
            if client.read(name) != value:
                raise MigrationError(f"Read-back comparison failed for {name}; stop before enabling sync.")
            print(f"CREATED_AND_VERIFIED {name}")
    print(f"{mode}: checked {len(source)} GitHub secret entries; {len(missing)} initially absent.")
    return len(source)


def main():
    try:
        if os.environ.get("GITHUB_EVENT_NAME") != "push":
            raise MigrationError("Run this through a push to the migration branch.")
        if os.environ.get("GITHUB_REF") != "refs/heads/chore/bac-370-infisical-import":
            raise MigrationError("Run this only from the dedicated migration branch.")
        manifest = json.loads(Path(__file__).with_name("infisical-secret-inventory.json").read_text())
        if os.environ.get("GITHUB_REPOSITORY") != manifest["repository"]:
            raise MigrationError("The repository does not match the inventory.")
        scope = os.environ.get("MIGRATION_SCOPE", "")
        if scope not in manifest["scopes"]:
            raise MigrationError("Unknown secret scope.")
        mode = os.environ.get("MIGRATION_MODE", "")
        if mode == "apply" and os.environ.get("SYNCS_CONFIRMED_DISABLED") != "true":
            raise MigrationError("Confirm the destination syncs are disabled before applying.")
        source = source_values(manifest["scopes"][scope], os.environ)
        client = InfisicalClient(
            os.environ.get("INFISICAL_MIGRATION_TOKEN", ""),
            scope,
            "/" + manifest["repository"].split("/")[1],
        )
        migrate(client, source, mode)
        if mode == "apply":
            migrate(client, source, "verify")
        return 0
    except MigrationError as error:
        print(f"ERROR: {error}", file=sys.stderr)
        return 1
    except Exception:
        # API bodies and unexpected exception messages can contain secret values.
        print("ERROR: Import stopped unexpectedly; details suppressed to protect secrets.", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
