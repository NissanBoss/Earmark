"""Downloads the home pages of the most visited sites, by the Tranco list,
with the scripts each loads from itself, for the calibration: how common
each identifier is on the web, and how many are on more than one site.

    python3 testdata/tranco.py /tmp/cal 2000

Run by calibrate.yml on GitHub's machines.
"""

import concurrent.futures
import io
import os
import re
import sys
import urllib.parse
import urllib.request
import zipfile

UA = {"User-Agent": "Mozilla/5.0 (compatible; earmark-calibration/1; +https://github.com/NissanBoss/Earmark)"}


def get(url, limit):
    with urllib.request.urlopen(urllib.request.Request(url, headers=UA), timeout=15) as r:
        return r.geturl(), r.read(limit)


def site(out, domain):
    try:
        final, html = get("https://%s/" % domain, 1_500_000)
    except Exception:
        return False
    dest = os.path.join(out, "tranco", domain)
    os.makedirs(dest, exist_ok=True)
    open(os.path.join(dest, "index.html"), "wb").write(html)
    base = urllib.parse.urlparse(final).netloc
    root = ".".join(base.split(".")[-2:])
    n = 0
    for src in re.findall(r'<script[^>]+src=["\']([^"\']+)["\']', html.decode("utf-8", "replace")):
        u = urllib.parse.urljoin(final, src)
        if not urllib.parse.urlparse(u).netloc.endswith(root) or n >= 6:
            continue
        try:
            _, body = get(u, 3_000_000)
        except Exception:
            continue
        n += 1
        open(os.path.join(dest, "script%d.js" % n), "wb").write(body)
    return True


def main():
    out, want = sys.argv[1], int(sys.argv[2])
    _, raw = get("https://tranco-list.eu/top-1m.csv.zip", 50_000_000)
    names = zipfile.ZipFile(io.BytesIO(raw)).read("top-1m.csv").decode().split()
    domains = [line.split(",")[1] for line in names[:want]]
    with concurrent.futures.ThreadPoolExecutor(16) as pool:
        done = sum(1 for ok in pool.map(lambda d: site(out, d), domains) if ok)
    print(done, "sites of", len(domains))


main()
