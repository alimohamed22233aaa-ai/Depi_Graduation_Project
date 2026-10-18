"""Search service for Online Boutique.

Loads the same products.json used by productcatalogservice and serves
full-text search over name, description and categories.
"""
import json
import os

from flask import Flask, jsonify, request
from prometheus_flask_exporter import PrometheusMetrics

PRODUCTS_FILE = os.environ.get("PRODUCTS_FILE", os.path.join(os.path.dirname(__file__), "products.json"))

app = Flask(__name__)
PrometheusMetrics(app)


def load_products(path=None):
    with open(path or PRODUCTS_FILE, encoding="utf-8") as f:
        raw = json.load(f)["products"]
    return [
        {
            "id": p["id"],
            "name": p["name"],
            "description": p["description"],
            "picture": p["picture"],
            "categories": p.get("categories", []),
            "price_usd": round(p["priceUsd"]["units"] + p["priceUsd"]["nanos"] / 1e9, 2),
        }
        for p in raw
    ]


PRODUCTS = load_products()


def matches(product, terms):
    haystack = " ".join([product["name"], product["description"], *product["categories"]]).lower()
    return all(t in haystack for t in terms)


@app.get("/health")
def health():
    return {"status": "ok", "service": "searchservice", "products": len(PRODUCTS)}


@app.get("/api/search")
def search():
    q = request.args.get("q", "").lower().split()
    category = request.args.get("category", "").lower()
    results = [p for p in PRODUCTS if matches(p, q)]
    if category:
        results = [p for p in results if category in p["categories"]]
    return jsonify({"query": request.args.get("q", ""), "count": len(results), "results": results})


PAGE = """<!doctype html><meta charset=utf-8><title>Boutique Search</title>
<style>body{font-family:system-ui;max-width:720px;margin:40px auto;padding:0 16px}
input{width:100%;padding:10px;font-size:16px}li{margin:10px 0}small{color:#666}</style>
<h2>Online Boutique search</h2><input id=q placeholder="try: watch, kitchen, tops" autofocus>
<ul id=r></ul>
<script>
const q=document.getElementById('q'),r=document.getElementById('r');
async function go(){const d=await (await fetch('/api/search?q='+encodeURIComponent(q.value))).json();
r.innerHTML=d.results.map(p=>`<li><b>${p.name}</b> - $${p.price_usd}<br><small>${p.description}</small></li>`).join('')}
q.oninput=go;go();
</script>"""


@app.get("/search")
def ui():
    return PAGE
