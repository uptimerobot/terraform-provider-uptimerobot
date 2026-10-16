import contextlib
import importlib.util
import io
import json
from pathlib import Path
import unittest
from unittest.mock import patch
from urllib.error import HTTPError, URLError

spec = importlib.util.spec_from_file_location("migration", Path(__file__).with_name("import-infisical-secrets.py"))
migration = importlib.util.module_from_spec(spec)
spec.loader.exec_module(migration)


class FakeClient:
    def __init__(self, existing=None):
        self.values = dict(existing or {})
        self.writes = []

    def read(self, name):
        return self.values.get(name)

    def create(self, name, value):
        self.writes.append((name, value))
        self.values[name] = value


class MigrationTests(unittest.TestCase):
    def test_http_json_preserves_multiline_signing_key_bytes(self):
        client = migration.InfisicalClient("dummy-auth-token", "repo", "/test")
        value = "-----BEGIN DUMMY KEY-----\n  dummy\r\n$literal\\suffix\n雪\n-----END DUMMY KEY-----\n"
        with patch.object(client.opener, "open", return_value=io.BytesIO(b'{}')) as request:
            client.create("GPG_PRIVATE_KEY", value)
        body = json.loads(request.call_args.args[0].data.decode("utf-8"))
        self.assertEqual(body["secretValue"].encode("utf-8"), value.encode("utf-8"))
        self.assertTrue(body["skipMultilineEncoding"])

    def run_migration(self, client, source, mode):
        with contextlib.redirect_stdout(io.StringIO()) as output:
            result = migration.migrate(client, source, mode)
        return result, output.getvalue()

    def test_dry_run_does_not_write_or_print_values(self):
        client = FakeClient()
        _, output = self.run_migration(client, {"TOKEN": "dummy-private-value"}, "dry-run")
        self.assertEqual(client.writes, [])
        self.assertIn("TOKEN", output)
        self.assertNotIn("dummy-private-value", output)

    def test_preserves_multiline_unicode_and_shell_characters_and_is_repeatable(self):
        value = "  dummy-雪\nsecond line\n' \\\n$literal & | \r\n"
        client = FakeClient()
        self.run_migration(client, {"TOKEN": value}, "apply")
        self.run_migration(client, {"TOKEN": value}, "apply")
        self.assertEqual(client.writes, [("TOKEN", value)])

    def test_conflict_is_found_before_any_write(self):
        client = FakeClient({"Z_TOKEN": "other-dummy"})
        with self.assertRaises(migration.MigrationError):
            self.run_migration(client, {"A_TOKEN": "new-dummy", "Z_TOKEN": "original-dummy"}, "apply")
        self.assertEqual(client.writes, [])

    def test_verify_requires_every_value_present(self):
        client = FakeClient()
        with self.assertRaises(migration.MigrationError):
            self.run_migration(client, {"TOKEN": "dummy"}, "verify")
        self.assertEqual(client.writes, [])

    def test_readback_mismatch_fails(self):
        client = FakeClient()
        client.create = lambda name, value: client.values.update({name: "corrupted-dummy"})
        with self.assertRaises(migration.MigrationError):
            self.run_migration(client, {"TOKEN": "original-dummy"}, "apply")

    def test_missing_source_fails_before_network(self):
        with self.assertRaises(migration.MigrationError):
            migration.source_values(["TOKEN"], {"SOURCE_TOKEN": ""})

    def test_only_manifest_names_are_read(self):
        self.assertEqual(
            migration.source_values(["TOKEN"], {"SOURCE_TOKEN": "dummy", "SOURCE_EXTRA": "excluded"}),
            {"TOKEN": "dummy"},
        )

    def test_invalid_mode_cannot_write(self):
        client = FakeClient()
        with self.assertRaises(migration.MigrationError):
            self.run_migration(client, {"TOKEN": "dummy"}, "unexpected")
        self.assertEqual(client.writes, [])

    def test_literal_infisical_references_are_rejected_before_any_write(self):
        client = FakeClient()
        with self.assertRaises(migration.MigrationError):
            self.run_migration(client, {"TOKEN": "${OTHER_SECRET}"}, "apply")
        self.assertEqual(client.writes, [])

    def test_redirection_cannot_forward_credentials(self):
        with self.assertRaises(migration.MigrationError):
            migration.NoRedirect().redirect_request(None, None, 302, "", {}, "https://example.invalid")

    def test_request_errors_never_include_response_body(self):
        client = migration.InfisicalClient("dummy-auth-token", "test", "/test")
        error = HTTPError("https://app.infisical.com", 403, "dummy-private-value", {}, io.BytesIO(b"dummy-private-value"))
        with patch.object(client.opener, "open", side_effect=error):
            with self.assertRaises(migration.MigrationError) as caught:
                client.read("TOKEN")
        self.assertNotIn("dummy-private-value", str(caught.exception))

    def test_network_errors_are_sanitized(self):
        client = migration.InfisicalClient("dummy-auth-token", "test", "/test")
        with patch.object(client.opener, "open", side_effect=URLError("dummy-private-value")):
            with self.assertRaises(migration.MigrationError) as caught:
                client.read("TOKEN")
        self.assertNotIn("dummy-private-value", str(caught.exception))

    def test_rate_limit_retries_after_the_requested_delay(self):
        client = migration.InfisicalClient("dummy-auth-token", "test", "/test")
        error = HTTPError("https://app.infisical.com", 429, "dummy-private-value", {"Retry-After": "5"}, io.BytesIO(b"dummy-private-value"))
        response = io.BytesIO(b'{"secret":{"secretKey":"TOKEN","secretValue":"dummy"}}')
        with patch.object(client.opener, "open", side_effect=[error, response]) as request, patch("time.sleep") as sleep:
            self.assertEqual(client.read("TOKEN"), "dummy")
        sleep.assert_called_once_with(5)
        self.assertEqual(request.call_count, 2)

    def test_rate_limit_retries_are_bounded_and_back_off(self):
        client = migration.InfisicalClient("dummy-auth-token", "test", "/test")
        error = HTTPError("https://app.infisical.com", 429, "dummy-private-value", {}, io.BytesIO(b"dummy-private-value"))
        with patch.object(client.opener, "open", side_effect=error) as request, patch("time.sleep") as sleep:
            with self.assertRaises(migration.MigrationError) as caught:
                client.read("TOKEN")
        self.assertEqual(request.call_count, 4)
        self.assertEqual([call.args[0] for call in sleep.call_args_list], [60, 120, 120])
        self.assertNotIn("dummy-private-value", str(caught.exception))

    def test_rate_limit_does_not_retry_before_a_long_requested_delay(self):
        client = migration.InfisicalClient("dummy-auth-token", "test", "/test")
        error = HTTPError("https://app.infisical.com", 429, "dummy-private-value", {"Retry-After": "600"}, io.BytesIO(b"dummy-private-value"))
        with patch.object(client.opener, "open", side_effect=error) as request, patch("time.sleep") as sleep:
            with self.assertRaises(migration.MigrationError):
                client.read("TOKEN")
        request.assert_called_once()
        sleep.assert_not_called()

    def test_create_retries_only_an_explicit_rate_limit_rejection(self):
        client = migration.InfisicalClient("dummy-auth-token", "test", "/test")
        error = HTTPError("https://app.infisical.com", 429, "dummy-private-value", {"Retry-After": "1"}, io.BytesIO(b"dummy-private-value"))
        with patch.object(client.opener, "open", side_effect=[error, io.BytesIO(b'{}')]) as request, patch("time.sleep"):
            client.create("TOKEN", "dummy-private-value")
        self.assertEqual(request.call_count, 2)
        self.assertEqual(request.call_args.args[0].method, "POST")
        with patch.object(client.opener, "open", side_effect=URLError("uncertain response")) as request, patch("time.sleep") as sleep:
            with self.assertRaises(migration.MigrationError):
                client.create("TOKEN", "dummy-private-value")
        request.assert_called_once()
        sleep.assert_not_called()

    def test_read_does_not_expand_references_or_use_imports(self):
        client = migration.InfisicalClient("dummy-auth-token", "test", "/test")
        response = {"secret": {"secretKey": "TOKEN", "secretValue": "dummy", "secretValueHidden": False}}
        with patch.object(client, "request", return_value=response) as request:
            self.assertEqual(client.read("TOKEN"), "dummy")
        params = request.call_args.kwargs["query"]
        self.assertEqual(params["expandSecretReferences"], "false")
        self.assertEqual(params["includeImports"], "false")
        self.assertEqual(params["type"], "shared")

    def test_hidden_value_is_not_treated_as_a_match(self):
        client = migration.InfisicalClient("dummy-auth-token", "test", "/test")
        with patch.object(client, "request", return_value={"secret": {"secretKey": "TOKEN", "secretValue": "***", "secretValueHidden": True}}):
            with self.assertRaises(migration.MigrationError):
                client.read("TOKEN")

    def test_create_preserves_newlines_and_does_not_update(self):
        client = migration.InfisicalClient("dummy-auth-token", "production", "/test")
        with patch.object(client, "request", return_value={}) as request:
            client.create("TOKEN", "dummy\n")
        self.assertEqual(request.call_args.args, ("POST", "TOKEN"))
        body = request.call_args.kwargs["body"]
        self.assertEqual(body["secretValue"], "dummy\n")
        self.assertTrue(body["skipMultilineEncoding"])



class WorkflowEntryTests(unittest.TestCase):
    def invoke(self, scope=None, mode="dry-run", overrides=None, client=None):
        manifest = json.loads(Path(__file__).with_name("infisical-secret-inventory.json").read_text())
        scope = scope or next(iter(manifest["scopes"]))
        environment = {
            "GITHUB_EVENT_NAME": "push",
            "GITHUB_REF": "refs/heads/chore/bac-370-infisical-import",
            "GITHUB_REPOSITORY": manifest["repository"],
            "MIGRATION_SCOPE": scope,
            "MIGRATION_MODE": mode,
            "SYNCS_CONFIRMED_DISABLED": "false",
            "INFISICAL_MIGRATION_TOKEN": "dummy-auth-token",
        }
        environment.update({"SOURCE_" + name: "dummy-value-" + name
                            for name in manifest["scopes"].get(scope, [])})
        environment.update(overrides or {})
        client = client or FakeClient()
        with (patch.dict(migration.os.environ, environment, clear=True),
              patch.object(migration, "InfisicalClient", return_value=client) as factory,
              contextlib.redirect_stdout(io.StringIO()) as output,
              contextlib.redirect_stderr(io.StringIO()) as errors):
            result = migration.main()
        return result, client, factory, output.getvalue() + errors.getvalue()

    def test_branch_push_dry_run_keeps_each_scope_separate_and_never_writes(self):
        manifest = json.loads(Path(__file__).with_name("infisical-secret-inventory.json").read_text())
        for scope in manifest["scopes"]:
            with self.subTest(scope=scope):
                result, client, factory, output = self.invoke(scope=scope)
                self.assertEqual(result, 0, output)
                factory.assert_called_once_with("dummy-auth-token", scope,
                                                "/" + manifest["repository"].split("/")[1])
                self.assertEqual(client.writes, [])
                self.assertNotIn("dummy-value-", output)
                self.assertNotIn("dummy-auth-token", output)

    def test_other_refs_are_rejected_before_accessing_infisical(self):
        for ref in ("refs/heads/develop", "refs/heads/main", "refs/pull/1/merge",
                    "refs/heads/chore/bac-370-infisical-import-other"):
            with self.subTest(ref=ref):
                result, _, factory, _ = self.invoke(overrides={"GITHUB_REF": ref})
                self.assertEqual(result, 1)
                factory.assert_not_called()

    def test_other_events_are_rejected_before_accessing_infisical(self):
        for event in ("workflow_dispatch", "pull_request", "pull_request_target", "schedule"):
            with self.subTest(event=event):
                result, _, factory, _ = self.invoke(overrides={"GITHUB_EVENT_NAME": event})
                self.assertEqual(result, 1)
                factory.assert_not_called()

    def test_other_repository_is_rejected_before_accessing_infisical(self):
        result, _, factory, _ = self.invoke(overrides={"GITHUB_REPOSITORY": "example/fork"})
        self.assertEqual(result, 1)
        factory.assert_not_called()

    def test_unknown_scope_is_rejected_before_accessing_infisical(self):
        for scope in ("unknown", "staging"):
            with self.subTest(scope=scope):
                result, _, factory, _ = self.invoke(scope=scope)
                self.assertEqual(result, 1)
                factory.assert_not_called()

    def test_apply_requires_explicit_sync_confirmation(self):
        result, client, factory, _ = self.invoke(mode="apply")
        self.assertEqual(result, 1)
        self.assertEqual(client.writes, [])
        factory.assert_not_called()

    def test_confirmed_apply_finishes_with_full_verification(self):
        with patch.object(migration, "migrate", wraps=migration.migrate) as run:
            result, client, _, output = self.invoke(
                mode="apply", overrides={"SYNCS_CONFIRMED_DISABLED": "true"})
        self.assertEqual(result, 0, output)
        self.assertTrue(client.writes)
        self.assertEqual([call.args[2] for call in run.call_args_list], ["apply", "verify"])
        self.assertNotIn("dummy-value-", output)

    def test_missing_source_is_rejected_before_accessing_infisical(self):
        manifest = json.loads(Path(__file__).with_name("infisical-secret-inventory.json").read_text())
        name = next(iter(manifest["scopes"].values()))[0]
        result, _, factory, _ = self.invoke(overrides={"SOURCE_" + name: ""})
        self.assertEqual(result, 1)
        factory.assert_not_called()


if __name__ == "__main__":
    unittest.main()
