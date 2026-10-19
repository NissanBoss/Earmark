"""Downloads personal sites from GitHub Pages for the calibration: the
user site of each owner, owner.github.io, and for as many as have one, a
second site of theirs, a project page at owner.github.io/repository. Two
sites of one person, and many people: what a tie between two of them is
worth can be counted, both ways.

    python3 testdata/sites.py /tmp/cal 1000

Run by calibrate.yml on GitHub's machines with GH_TOKEN set. Each site is
its home page and the scripts it loads from itself, which is what anybody
who opens it gets.
"""

import concurrent.futures
import json
import os
import re
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


def get(url, limit):
    with urllib.request.urlopen(urllib.request.Request(url, headers=UA), timeout=20) as r:
        return r.read(limit)


def save_site(url, dest):
    try:
        html = get(url, 1_000_000)
    except Exception:
        return False
    os.makedirs(dest, exist_ok=True)
    open(os.path.join(dest, "index.html"), "wb").write(html)
    host = urllib.parse.urlparse(url).netloc
    n = 0
    for src in re.findall(r'<script[^>]+src=["\']([^"\']+)["\']', html.decode("utf-8", "replace")):
        u = urllib.parse.urljoin(url, src)
        if urllib.parse.urlparse(u).netloc != host or n >= 6:
            continue
        try:
            body = get(u, 2_000_000)
        except Exception:
            continue
        n += 1
        open(os.path.join(dest, "script%d.js" % n), "wb").write(body)
    return True


def owners(want):
    seen = []
    # The search gives a thousand results a query at most, so the query is
    # cut by the month the repository was made.
    months = ["%d-%02d" % (y, m) for y in range(2026, 2014, -1) for m in range(12, 0, -1)]
    for month in months:
        for page in range(1, 4):
            q = urllib.parse.quote("github.io in:name created:%s-01..%s-28" % (month, month))
            try:
                items = api("/search/repositories?q=%s&per_page=100&page=%d" % (q, page)).get("items", [])
            except Exception:
                break
            for it in items:
                owner = it["owner"]["login"]
                if it["name"].lower() == owner.lower() + ".github.io" and owner not in seen:
                    seen.append(owner)
            if len(items) < 100 or len(seen) >= want * 2:
                break
        if len(seen) >= want * 2:
            break
    return seen


def one(out, owner, pages):
    if not save_site("https://%s.github.io/" % owner.lower(), os.path.join(out, "sites", owner)):
        return 0
    if not pages:
        return 1
    try:
        repos = api("/users/%s/repos?per_page=100&sort=updated" % owner)
    except Exception:
        return 1
    for r in repos:
        if r.get("has_pages") and r["name"].lower() != owner.lower() + ".github.io" and not r.get("fork"):
            if save_site("https://%s.github.io/%s/" % (owner.lower(), r["name"]), os.path.join(out, "projects", owner)):
                return 2
    return 1


def main():
    out, want = sys.argv[1], int(sys.argv[2])
    candidates = owners(want)[: want * 3 // 2]
    done = 0
    pairs = 0
    # The token a workflow has is good for a thousand calls an hour, so
    # second sites are looked for among the first five hundred owners only.
    with concurrent.futures.ThreadPoolExecutor(8) as pool:
        for got in pool.map(lambda io: one(out, io[1], io[0] < 500), enumerate(candidates)):
            if got:
                done += 1
            if got == 2:
                pairs += 1
            if done >= want:
                break
    print("%d sites, %d owners with a second one" % (done, pairs))


main()
