#!/usr/bin/env python3
import http.cookiejar
import json
import os
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path


SMTP_KEYS = {
    "EmailVerificationEnabled",
    "SMTPAccount",
    "SMTPForceAuthLogin",
    "SMTPFrom",
    "SMTPInsecureSkipVerify",
    "SMTPPort",
    "SMTPSSLEnabled",
    "SMTPServer",
    "SMTPStartTLSEnabled",
    "SMTPToken",
}


def request(opener, base_url, method, path, payload=None, headers=None):
    body = None
    request_headers = {"Content-Type": "application/json"}
    if headers:
        request_headers.update(headers)
    if payload is not None:
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(
        base_url.rstrip("/") + path,
        data=body,
        headers=request_headers,
        method=method,
    )
    with opener.open(req, timeout=15) as response:
        result = json.loads(response.read().decode("utf-8"))
    if result.get("success") is not True:
        raise RuntimeError(result.get("message") or f"New API 请求失败：{path}")
    return result.get("data")


def wait_ready(opener, base_url):
    last_error = None
    for _ in range(60):
        try:
            return request(opener, base_url, "GET", "/api/setup")
        except (OSError, urllib.error.URLError, ValueError, RuntimeError) as error:
            last_error = error
            time.sleep(1)
    raise RuntimeError(f"New API 初始化入口未就绪：{last_error}")


def load_smtp_options(path):
    values = json.loads(Path(path).read_text(encoding="utf-8"))
    if not isinstance(values, dict):
        raise RuntimeError("SMTP 备份格式无效")
    unknown = set(values) - SMTP_KEYS
    if unknown:
        raise RuntimeError("SMTP 备份包含未批准设置键")
    return {key: str(value) for key, value in values.items()}


def main():
    if len(sys.argv) != 3:
        raise SystemExit("用法：bootstrap-newapi.py <New API 地址> <SMTP 设置备份>")
    username = os.environ.get("NEW_API_ADMIN_USERNAME", "").strip()
    password = os.environ.get("NEW_API_ADMIN_PASSWORD", "")
    if not username or not password:
        raise SystemExit("缺少 New API 管理员环境变量")

    base_url, smtp_path = sys.argv[1], sys.argv[2]
    opener = urllib.request.build_opener(
        urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar())
    )
    setup = wait_ready(opener, base_url)
    if not setup.get("status"):
        request(
            opener,
            base_url,
            "POST",
            "/api/setup",
            {
                "username": username,
                "password": password,
                "confirmPassword": password,
                "SelfUseModeEnabled": False,
                "DemoSiteEnabled": False,
            },
        )

    login = request(
        opener,
        base_url,
        "POST",
        "/api/user/login",
        {"username": username, "password": password},
    )
    if int(login.get("role", 0)) != 100 or int(login.get("id", 0)) <= 0:
        raise RuntimeError("New API 唯一管理员身份校验失败")
    headers = {"New-Api-User": str(login["id"])}
    for key, value in load_smtp_options(smtp_path).items():
        request(
            opener,
            base_url,
            "PUT",
            "/api/option/",
            {"key": key, "value": value},
            headers,
        )
    print("New API 唯一管理员与 SMTP 设置恢复完成。")


if __name__ == "__main__":
    main()
