import os
import sys
sys.path.insert(0, os.path.dirname(__file__))
import pytest  # noqa: E402
from app import create_app  # noqa: E402

ORDER = {"order_id": "o-1", "user_id": "u-1", "email": "a@b.com", "tracking_id": "T1",
         "currency": "USD", "total": 42.5, "items": [{"product_id": "OLJCESPC7Z", "quantity": 2}]}


@pytest.fixture()
def client(tmp_path):
    return create_app(f"sqlite:///{tmp_path}/t.db").test_client()


def test_health(client):
    assert client.get("/health").status_code == 200


def test_create_and_get(client):
    assert client.post("/api/orders", json=ORDER).status_code == 201
    r = client.get("/api/orders/o-1").get_json()
    assert r["total"] == 42.5 and r["items"][0]["quantity"] == 2


def test_duplicate_rejected(client):
    client.post("/api/orders", json=ORDER)
    assert client.post("/api/orders", json=ORDER).status_code == 409


def test_validation(client):
    assert client.post("/api/orders", json={"user_id": "x"}).status_code == 400


def test_list_filter_by_user(client):
    client.post("/api/orders", json=ORDER)
    client.post("/api/orders", json={**ORDER, "order_id": "o-2", "user_id": "u-2"})
    assert len(client.get("/api/orders").get_json()) == 2
    assert len(client.get("/api/orders?user_id=u-2").get_json()) == 1


def test_not_found(client):
    assert client.get("/api/orders/nope").status_code == 404
