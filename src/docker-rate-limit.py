#!/usr/bin/python3

from kubernetes import client, config
import base64
import inspect
import threading
import time
import requests
import json
import os
from flask import Flask, Response

docker_token_url = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:ratelimitpreview/test:pull"
docker_rate_limit_url = (
    "https://registry-1.docker.io/v2/ratelimitpreview/test/manifests/latest"
)
timeout_sec = 5
refresh_seconds = 300
port_number = 9902
imagepull_secret = ""
docker_source = ""
docker_limit, docker_remaining = 0, 0
initialized = False

app = Flask(__name__)


def get_env():
    global refresh_seconds, port_number, imagepull_secret, standalone, namespace

    strval = os.environ.get("REFRESH_SECONDS", "300")
    try:
        refresh_seconds = int(strval)
    except ValueError:
        print(
            "environment variable REFRESH_SECONDS=%s, expecting int" % (strval),
            flush=True,
        )
        refresh_seconds = 300

    strval = os.environ.get("PORT_NUMBER", "9902")
    try:
        port_number = int(strval)
    except ValueError:
        print(
            "environment variable PORT_NUMBER=%s, expecting int" % (strval), flush=True
        )
        port_number = 9900

    imagepull_secret = os.environ.get("IMAGEPULL_SECRET", "")
    if imagepull_secret == "":
        print("environment variable IMAGEPULL_SECRET not set", flush=True)

    standalone = os.environ.get("STANDALONE")
    if standalone is None:
        print("environment variable STANDALONE not set", flush=True)
        standalone = 0
    else:
        standalone = 1

    namespace = os.environ.get("NAMESPACE")
    if namespace == "":
        print("environment variable NAMESPACE not set", flush=True)

    print("environment REFRESH_SECONDS: %s" % (refresh_seconds), flush=True)
    print("environment PORT_NUMBER: %s" % (port_number), flush=True)
    print("environment IMAGEPULL_SECRET: %s" % (imagepull_secret), flush=True)
    print("environment STANDALONE: %s" % (standalone), flush=True)
    print("environment NAMESPACE: %s" % (namespace), flush=True)


def get_secret(name: str, namespace: str) -> (str, str):
    global standalone

    if standalone == 1:
        config.load_kube_config()
    else:
        config.load_incluster_config()

    api_instance = client.CoreV1Api()
    try:
        api_response = api_instance.read_namespaced_secret(
            name, namespace, pretty="false"
        )
    except client.exceptions.ApiException as ex:
        print(
            "get_secret %s/%s exception %s" % (namespace, imagepull_secret, ex),
            flush=True,
        )
        exit(1)

    dockerconfigjson = json.loads(
        base64.b64decode(api_response.data[".dockerconfigjson"])
    )
    try:
        user = dockerconfigjson["auths"]["https://index.docker.io/v1/"]["username"]
        passwd = dockerconfigjson["auths"]["https://index.docker.io/v1/"]["password"]
        return user, passwd
    except KeyError:
        print("wrong type of secret", flush=True)
        exit(1)


def docker_token() -> str:
    global imagepull_secret, docker_source

    try:
        if imagepull_secret == "":
            docker_source = "anonymous"
            resp = requests.get(docker_token_url, timeout=timeout_sec)
        else:
            user, password = get_secret(imagepull_secret, namespace)
            docker_source = user
            resp = requests.get(
                docker_token_url, auth=(user, password), timeout=timeout_sec
            )
    except Exception as e:
        print("exception on request " + str(e), flush=True)
        return ""

    if resp.status_code != 200:
        print("docker_token status code %s" % (resp.status_code), flush=True)
        return ""

    json_content = json.loads(resp.content)
    return json_content["token"]


def get_int_from_reply(reply: str) -> int:
    string = reply.split(";")[0]
    try:
        return int(string)
    except (ValueError, TypeError):
        print("get_int_from_reply: %s not a number" % (string), flush=True)
        return -1


def docker_limits(token: str) -> (int, int):
    headers = {"Content-Type": "application/json", "Authorization": "Bearer " + token}

    try:
        resp = requests.head(
            docker_rate_limit_url, headers=headers, timeout=timeout_sec
        )
    except Exception as e:
        print("exception on request " + str(e), flush=True)
        return -1, -1, ""

    if resp.status_code != 200:
        print("docker_limits status code %s" % (resp.status_code), flush=True)
        return -1, -1, ""

    # "ratelimit-remaining" and "ratelimit-limit" are 2 strings in the form: '77;w=21600'
    # docker-ratelimit-source: 94.209.190.149 or docker-ratelimit-source: 63bc4b48-54a6-4ff2-9e74-7c7011a33719
    try:
        remaining_str = resp.headers["ratelimit-remaining"]
        limit_str = resp.headers["ratelimit-limit"]
        source = resp.headers["docker-ratelimit-source"]
    except KeyError as e:
        print("docker_limits: key not found %s" % (e))
        return -1, -1, ""

    remaining = get_int_from_reply(remaining_str)
    limit = get_int_from_reply(limit_str)
    print(
        "docker_limits: remaining %s, limit %s source %s" % (remaining, limit, source),
        flush=True,
    )
    return remaining, limit, source


def docker_stats():
    global docker_limit, docker_remaining, docker_source, initialized

    while True:
        token = docker_token()
        remaining, limit, source = docker_limits(token)
        if remaining != -1 and limit != -1 and source != "":
            # all valid values
            docker_limit, docker_remaining = limit, remaining
            if docker_source == "anonymous":
                docker_source = source
            initialized = True
        time.sleep(refresh_seconds)


@app.route("/health", methods=["GET"])
def show_health() -> (str, int):
    if docker_thread.is_alive():
        return "healthy", 200
    else:
        return "not healthy", 404


@app.route("/metrics", methods=["GET"])
def show_metrics() -> (str, int):
    global docker_limit, docker_remaining, docker_source, initialized

    if not initialized:
        return "not initialized", 404

    metrics_str = inspect.cleandoc(
        """
        # HELP ratelimit_limit docker hub ratelimit
        # TYPE ratelimit_limit gauge
        # HELP ratelimit_remaining docker hub ratelimit remaining
        # TYPE ratelimit_remaining gauge
        ratelimit_limit{account="%s"} %s
        ratelimit_remaining{account="%s"} %s
        """
        % (docker_source, docker_limit, docker_source, docker_remaining)
    )

    print(
        "show_metrics: ratelimit_limit %s, ratelimit_remaining %s source %s"
        % (docker_limit, docker_remaining, docker_source),
        flush=True,
    )
    return Response(metrics_str, mimetype="text/plain")


def main() -> None:
    global docker_thread

    get_env()

    docker_thread = threading.Thread(target=docker_stats)
    docker_thread.start()

    app.run(host="0.0.0.0", port=port_number)


if __name__ == "__main__":
    main()
