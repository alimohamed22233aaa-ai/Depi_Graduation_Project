import os
import sys
sys.path.insert(0, os.path.dirname(__file__))
import app as search_app  # noqa: E402

client = search_app.app.test_client()


def test_health():
    r = client.get("/health")
    assert r.status_code == 200 and r.get_json()["products"] > 0


def test_search_by_name():
    r = client.get("/api/search?q=sunglasses").get_json()
    assert r["count"] >= 1 and r["results"][0]["name"] == "Sunglasses"


def test_search_no_match():
    assert client.get("/api/search?q=zzzzqqq").get_json()["count"] == 0


def test_category_filter():
    r = client.get("/api/search?category=accessories").get_json()
    assert r["count"] >= 1
    assert all("accessories" in p["categories"] for p in r["results"])


def test_ui_page():
    assert b"Boutique" in client.get("/search").data
