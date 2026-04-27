#!/bin/sh
set -eu

if [ -z "${VLESS_URL:-}" ]; then
  echo "VLESS_URL is required" >&2
  exit 1
fi

python3 - <<'PY'
import json
import os
import sys
from urllib.parse import parse_qs, unquote, urlparse

raw_url = os.environ["VLESS_URL"]
log_level = os.environ.get("XRAY_LOG_LEVEL", "warning")
parsed = urlparse(raw_url)

if parsed.scheme != "vless":
    sys.exit("VLESS_URL must start with vless://")
if not parsed.username:
    sys.exit("VLESS_URL must include a user id")
if not parsed.hostname:
    sys.exit("VLESS_URL must include a host")
if not parsed.port:
    sys.exit("VLESS_URL must include a port")

query = {key: values[-1] for key, values in parse_qs(parsed.query).items()}

stream = query.get("type", "tcp")
security = query.get("security", "none")
network_settings = {}

if stream == "tcp":
    header_type = query.get("headerType", "none")
    network_settings = {"header": {"type": header_type}}
elif stream == "ws":
    network_settings = {
        "path": query.get("path", "/"),
        "headers": {"Host": query.get("host", parsed.hostname)},
    }
elif stream == "grpc":
    network_settings = {"serviceName": query.get("serviceName", "")}
elif stream == "httpupgrade":
    network_settings = {
        "path": query.get("path", "/"),
        "host": query.get("host", parsed.hostname),
    }
elif stream == "splithttp":
    network_settings = {
        "path": query.get("path", "/"),
        "host": query.get("host", parsed.hostname),
    }
else:
    sys.exit(f"Unsupported VLESS transport type: {stream}")

stream_settings = {
    "network": stream,
    "security": security,
    f"{stream}Settings": network_settings,
}

if security == "tls":
    stream_settings["tlsSettings"] = {
        "serverName": query.get("sni", query.get("host", parsed.hostname)),
        "allowInsecure": query.get("allowInsecure", "0") in ("1", "true", "TRUE"),
    }
    if query.get("alpn"):
        stream_settings["tlsSettings"]["alpn"] = [item for item in query["alpn"].split(",") if item]
elif security == "reality":
    stream_settings["realitySettings"] = {
        "serverName": query.get("sni", parsed.hostname),
        "publicKey": query.get("pbk", ""),
        "shortId": query.get("sid", ""),
        "fingerprint": query.get("fp", "chrome"),
        "spiderX": unquote(query.get("spx", "/")),
    }
elif security != "none":
    sys.exit(f"Unsupported VLESS security: {security}")

config = {
    "log": {"loglevel": log_level},
    "inbounds": [
        {
            "tag": "socks-in",
            "listen": "0.0.0.0",
            "port": 1080,
            "protocol": "socks",
            "settings": {"udp": True, "auth": "noauth"},
        }
    ],
    "outbounds": [
        {
            "tag": "proxy",
            "protocol": "vless",
            "settings": {
                "vnext": [
                    {
                        "address": parsed.hostname,
                        "port": parsed.port,
                        "users": [
                            {
                                "id": parsed.username,
                                "encryption": query.get("encryption", "none"),
                                "flow": query.get("flow", ""),
                            }
                        ],
                    }
                ]
            },
            "streamSettings": stream_settings,
        },
        {"tag": "direct", "protocol": "freedom"},
        {"tag": "block", "protocol": "blackhole"},
    ],
}

os.makedirs("/etc/xray", exist_ok=True)
with open("/etc/xray/config.json", "w", encoding="utf-8") as config_file:
    json.dump(config, config_file, indent=2)
PY

exec xray run -config /etc/xray/config.json
