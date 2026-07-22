import json
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


def usage():
    return {
        "prompt_tokens": 10,
        "completion_tokens": 5,
        "total_tokens": 15,
        "prompt_tokens_details": {"cached_tokens": 4},
    }


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, _format, *_args):
        return

    def send_json(self, payload, status=200):
        body = json.dumps(payload, separators=(",", ":")).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path == "/health":
            self.send_json({"status": "ok"})
            return
        if self.path == "/v1/models":
            self.send_json({"object": "list", "data": [{"id": "fixture-model", "object": "model"}]})
            return
        self.send_json({"error": {"message": "not found", "type": "fixture_error"}}, 404)

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        request = json.loads(self.rfile.read(length) or b"{}")
        model = request.get("model", "fixture-model")
        if model != "fixture-model":
            self.send_json({"error": {"message": "model mapping missing", "type": "fixture_error"}}, 400)
            return
        if self.path == "/v1/chat/completions":
            if request.get("stream"):
                self.send_response(200)
                self.send_header("Content-Type", "text/event-stream")
                self.send_header("Cache-Control", "no-cache")
                self.send_header("Connection", "close")
                self.end_headers()
                chunks = [
                    {
                        "id": "chatcmpl-stream-fixture",
                        "object": "chat.completion.chunk",
                        "created": int(time.time()),
                        "model": model,
                        "choices": [{"index": 0, "delta": {"role": "assistant", "content": "fixture "}, "finish_reason": None}],
                    },
                    {
                        "id": "chatcmpl-stream-fixture",
                        "object": "chat.completion.chunk",
                        "created": int(time.time()),
                        "model": model,
                        "choices": [{"index": 0, "delta": {"content": "stream"}, "finish_reason": "stop"}],
                        "usage": usage(),
                    },
                ]
                for chunk in chunks:
                    self.wfile.write(b"data: " + json.dumps(chunk, separators=(",", ":")).encode() + b"\n\n")
                    self.wfile.flush()
                self.wfile.write(b"data: [DONE]\n\n")
                self.wfile.flush()
                return
            self.send_json(
                {
                    "id": "chatcmpl-fixture",
                    "object": "chat.completion",
                    "created": int(time.time()),
                    "model": model,
                    "choices": [{"index": 0, "message": {"role": "assistant", "content": "fixture chat"}, "finish_reason": "stop"}],
                    "usage": usage(),
                }
            )
            return
        if self.path == "/v1/responses":
            self.send_json(
                {
                    "id": "resp_fixture",
                    "object": "response",
                    "created_at": int(time.time()),
                    "status": "completed",
                    "model": model,
                    "output": [
                        {
                            "id": "msg_fixture",
                            "type": "message",
                            "status": "completed",
                            "role": "assistant",
                            "content": [{"type": "output_text", "text": "fixture response", "annotations": []}],
                        }
                    ],
                    "usage": {
                        "input_tokens": 10,
                        "input_tokens_details": {"cached_tokens": 4},
                        "output_tokens": 5,
                        "output_tokens_details": {"reasoning_tokens": 0},
                        "total_tokens": 15,
                    },
                }
            )
            return
        if self.path == "/v1/embeddings":
            self.send_json(
                {
                    "object": "list",
                    "data": [{"object": "embedding", "embedding": [0.125, -0.25, 0.5], "index": 0}],
                    "model": model,
                    "usage": {"prompt_tokens": 10, "total_tokens": 10},
                }
            )
            return
        self.send_json({"error": {"message": "not found", "type": "fixture_error"}}, 404)


ThreadingHTTPServer(("0.0.0.0", 8081), Handler).serve_forever()
