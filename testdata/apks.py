"""Downloads Android apps that their developers publish on GitHub, from the
latest release of each, for the calibration: what their signing
certificates say, and how many developers sign apps under more than one
account with the same key.

    python3 testdata/apks.py /tmp/cal 200

Run by calibrate.yml on GitHub's machines with GH_TOKEN set. F-Droid's apps
are not used: F-Droid signs most of them with its own keys.
"""

import json
import os
import sys
import urllib.parse
import urllib.request

UA = {"User-Agent": "earmark-calibration/1 (https://github.com/NissanBoss/Earmark)"}
TOKEN = os.environ.get("GH_TOKEN", "")


def api(path):
    h = dict(UA, Accept="application/vnd.github+json")
    if TOKEN:
        h["Authorization"] = "Bearer " + TOKEN
    with urllib.request.urlopen(urllib.request.Request("https://api.github.com" + path, headers=h), timeout=30) as r:
        return json.loads(r.read())


def main():
    out, want = os.path.join(sys.argv[1], "apks"), int(sys.argv[2])
    os.makedirs(out, exist_ok=True)
    got = 0
    for page in range(1, 11):
        q = urllib.parse.quote("topic:android language:kotlin stars:>20")
        try:
            items = api("/search/repositories?q=%s&sort=updated&per_page=100&page=%d" % (q, page)).get("items", [])
        except Exception:
            break
        for it in items:
            try:
                rel = api("/repos/%s/releases/latest" % it["full_name"])
            except Exception:
                continue
            apks = [a for a in rel.get("assets", []) if a["name"].endswith(".apk") and a["size"] < 60_000_000]
            if not apks:
                continue
            a = min(apks, key=lambda a: a["size"])
            dest = os.path.join(out, it["full_name"].replace("/", "__") + ".apk")
            try:
                with urllib.request.urlopen(urllib.request.Request(a["browser_download_url"], headers=UA), timeout=120) as r:
                    open(dest, "wb").write(r.read())
            except Exception:
                continue
            got += 1
            if got >= want:
                print(got, "apps")
                return
    print(got, "apps")


main()
