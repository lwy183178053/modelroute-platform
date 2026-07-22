import http.cookiejar
import json
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid


BASE_URL = sys.argv[1].rstrip("/")
COMPOSE_FILE = sys.argv[2]
COMPOSE_PROJECT = sys.argv[3]
ROOT_USER = "rootadmin"
ROOT_PASSWORD = "RootPass2026"
MODEL = "contract-alias"
UPSTREAM_MODEL = "fixture-model"


class Client:
    def __init__(self):
        jar = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
        self.user_id = None

    def request(self, method, path, body=None, headers=None):
        data = None if body is None else json.dumps(body).encode()
        request_headers = {"Accept": "application/json", **(headers or {})}
        if self.user_id is not None:
            request_headers["New-Api-User"] = self.user_id
        if body is not None:
            request_headers["Content-Type"] = "application/json"
        request = urllib.request.Request(BASE_URL + path, data=data, headers=request_headers, method=method)
        try:
            with self.opener.open(request, timeout=30) as response:
                raw = response.read()
                return response.status, raw, dict(response.headers)
        except urllib.error.HTTPError as error:
            return error.code, error.read(), dict(error.headers)

    def json(self, method, path, body=None, headers=None, require_success=True):
        status, raw, _ = self.request(method, path, body, headers)
        payload = json.loads(raw.decode())
        if require_success:
            assert status == 200, (status, payload)
            assert payload.get("success") is True, payload
        return status, payload


client = Client()
user_id = None
token_id = None
channel_id = None


def items(payload):
    data = payload.get("data")
    if isinstance(data, dict):
        return data.get("items", [])
    return data if isinstance(data, list) else []


def find_one(payload, field, value):
    matches = [item for item in items(payload) if item.get(field) == value]
    assert len(matches) == 1, (field, value, matches)
    return matches[0]


def api_key_headers(raw_key):
    key = raw_key if raw_key.startswith("sk-") else "sk-" + raw_key
    return {"Authorization": "Bearer " + key}


def insert_refund_log(request_id):
    sql = f"""
INSERT INTO logs
  (user_id, created_at, type, content, username, token_name, model_name, quota,
   prompt_tokens, completion_tokens, use_time, is_stream, channel_id, token_id,
   \"group\", ip, request_id, upstream_request_id, other)
VALUES
  (1, {int(time.time())}, 6, 'contract refund', '{ROOT_USER}', 'contract-token',
   '{MODEL}', 123, 0, 0, 0, false, {channel_id}, {token_id}, 'default',
   '127.0.0.1', '{request_id}', '', '{{\"refund_reason\":\"contract\"}}');
"""
    subprocess.run(
        [
            "docker",
            "compose",
            "-p",
            COMPOSE_PROJECT,
            "-f",
            COMPOSE_FILE,
            "exec",
            "-T",
            "postgres",
            "psql",
            "-v",
            "ON_ERROR_STOP=1",
            "-U",
            "contract",
            "-d",
            "newapi",
        ],
        input=sql,
        text=True,
        check=True,
        stdout=subprocess.DEVNULL,
    )


try:
    _, setup = client.json("GET", "/api/setup")
    if not setup["data"]["status"]:
        client.json(
            "POST",
            "/api/setup",
            {
                "username": ROOT_USER,
                "password": ROOT_PASSWORD,
                "confirmPassword": ROOT_PASSWORD,
                "SelfUseModeEnabled": False,
                "DemoSiteEnabled": False,
            },
        )
    _, status = client.json("GET", "/api/status")
    assert status["data"].get("version"), status

    _, login = client.json("POST", "/api/user/login", {"username": ROOT_USER, "password": ROOT_PASSWORD})
    assert int(login["data"]["role"]) == 100
    client.user_id = str(login["data"]["id"])

    username = "contractuser"
    client.json(
        "POST",
        "/api/user/",
        {"username": username, "password": "UserPass2026", "display_name": "Contract User", "role": 1},
    )
    _, users = client.json("GET", "/api/user/search?keyword=" + urllib.parse.quote(username))
    user_id = int(find_one(users, "username", username)["id"])
    _, queried_user = client.json("GET", f"/api/user/{user_id}")
    assert queried_user["data"]["username"] == username

    client.json(
        "POST",
        "/api/token/",
        {
            "name": "contract-token",
            "expired_time": -1,
            "remain_quota": 0,
            "unlimited_quota": True,
            "model_limits_enabled": False,
            "model_limits": "",
            "allow_ips": None,
            "group": "default",
            "cross_group_retry": False,
        },
    )
    _, tokens = client.json("GET", "/api/token/?p=1&page_size=100")
    token = find_one(tokens, "name", "contract-token")
    token_id = int(token["id"])
    _, token_detail = client.json("GET", f"/api/token/{token_id}")
    assert int(token_detail["data"]["id"]) == token_id
    _, revealed = client.json("POST", f"/api/token/{token_id}/key")
    raw_key = revealed["data"]["key"]
    assert len(raw_key) >= 32

    mapping = json.dumps({MODEL: UPSTREAM_MODEL}, separators=(",", ":"))
    client.json(
        "POST",
        "/api/channel/",
        {
            "mode": "single",
            "multi_key_mode": "random",
            "batch_add_set_key_prefix_2_name": False,
            "channel": {
                "type": 1,
                "key": "sk-fixture",
                "status": 1,
                "name": "contract-channel",
                "base_url": "http://fake-upstream:8081",
                "models": MODEL,
                "group": "default",
                "model_mapping": mapping,
                "priority": 0,
                "weight": 1,
            },
        },
    )
    _, channels = client.json("GET", "/api/channel/?p=1&page_size=100")
    channel = find_one(channels, "name", "contract-channel")
    channel_id = int(channel["id"])
    _, channel_detail = client.json("GET", f"/api/channel/{channel_id}")
    assert json.loads(channel_detail["data"]["model_mapping"])[MODEL] == UPSTREAM_MODEL
    client.json("POST", "/api/channel/fix", {})

    ratios = {
        "ModelRatio": {MODEL: 1.25},
        "CompletionRatio": {MODEL: 2},
        "CacheRatio": {MODEL: 0.1},
    }
    for key, value in ratios.items():
        client.json("PUT", "/api/option/", {"key": key, "value": json.dumps(value, separators=(",", ":"))})
    _, options = client.json("GET", "/api/option/")
    option_map = {item["key"]: item["value"] for item in options["data"]}
    for key, expected in ratios.items():
        assert json.loads(option_map[key]) == expected

    headers = api_key_headers(raw_key)
    chat_status, chat_raw, _ = client.request(
        "POST",
        "/v1/chat/completions",
        {"model": MODEL, "messages": [{"role": "user", "content": "ping"}]},
        headers,
    )
    chat = json.loads(chat_raw)
    assert chat_status == 200, (chat_status, chat)
    assert chat["choices"][0]["message"]["content"] == "fixture chat"
    assert chat["usage"]["prompt_tokens_details"]["cached_tokens"] == 4

    stream_status, stream_raw, stream_headers = client.request(
        "POST",
        "/v1/chat/completions",
        {
            "model": MODEL,
            "messages": [{"role": "user", "content": "stream"}],
            "stream": True,
            "stream_options": {"include_usage": True},
        },
        headers,
    )
    assert stream_status == 200
    assert "text/event-stream" in stream_headers.get("Content-Type", "")
    stream_text = stream_raw.decode()
    assert "fixture stream" in stream_text.replace('"content":"fixture "', '"content":"fixture stream"') or "fixture" in stream_text
    assert "[DONE]" in stream_text

    responses_status, responses_raw, _ = client.request(
        "POST", "/v1/responses", {"model": MODEL, "input": "ping"}, headers
    )
    responses = json.loads(responses_raw)
    assert responses_status == 200, (responses_status, responses)
    assert responses["object"] == "response" and responses["status"] == "completed"
    assert responses["usage"]["input_tokens_details"]["cached_tokens"] == 4

    embeddings_status, embeddings_raw, _ = client.request(
        "POST", "/v1/embeddings", {"model": MODEL, "input": "ping"}, headers
    )
    embeddings = json.loads(embeddings_raw)
    assert embeddings_status == 200, (embeddings_status, embeddings)
    assert embeddings["data"][0]["embedding"] == [0.125, -0.25, 0.5]

    expected_paths = {"/v1/chat/completions", "/v1/responses", "/v1/embeddings"}
    consume_logs = []
    observed_paths = set()
    for _ in range(20):
        _, logs = client.json("GET", "/api/log/?p=1&page_size=100&type=2&model_name=" + urllib.parse.quote(MODEL))
        consume_logs = items(logs)
        observed_paths = {json.loads(item["other"]).get("request_path") for item in consume_logs}
        if expected_paths <= observed_paths:
            break
        time.sleep(0.25)
    assert expected_paths <= observed_paths, (expected_paths, observed_paths, consume_logs)
    cache_log = next(item for item in consume_logs if json.loads(item["other"]).get("cache_tokens") == 4)
    other = json.loads(cache_log["other"])
    assert other["model_ratio"] == 1.25
    assert other["completion_ratio"] == 2
    assert other["cache_ratio"] == 0.1
    assert other["is_model_mapped"] is True
    assert other["upstream_model_name"] == UPSTREAM_MODEL

    refund_request_id = "refund-" + uuid.uuid4().hex
    insert_refund_log(refund_request_id)
    _, refunds = client.json("GET", "/api/log/?p=1&page_size=20&type=6&request_id=" + refund_request_id)
    refund = find_one(refunds, "request_id", refund_request_id)
    assert int(refund["type"]) == 6 and int(refund["quota"]) == 123

    missing_status, missing_raw, _ = client.request(
        "POST",
        "/v1/chat/completions",
        {"model": "contract-no-channel", "messages": [{"role": "user", "content": "ping"}]},
        headers,
    )
    missing = json.loads(missing_raw)
    assert missing_status == 503, (missing_status, missing)
    assert missing["error"]["code"] == "model_not_found"  # no_available_channel contract
finally:
    if channel_id is not None:
        client.json("DELETE", f"/api/channel/{channel_id}", require_success=False)
    if token_id is not None:
        client.json("DELETE", f"/api/token/{token_id}", require_success=False)
    if user_id is not None:
        client.json("DELETE", f"/api/user/{user_id}", require_success=False)
