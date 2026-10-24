import demo_pb2
from grpc_health.v1 import health_pb2

import recommendation_server


class FakeCatalogStub:
    """Pretends to be the product catalog service."""

    def __init__(self, product_ids):
        self.product_ids = product_ids

    def ListProducts(self, request):
        return demo_pb2.ListProductsResponse(
            products=[demo_pb2.Product(id=pid) for pid in self.product_ids]
        )


def make_service(monkeypatch, catalog_ids):
    # product_catalog_stub is only created when the server starts,
    # so we inject a fake one for the test.
    monkeypatch.setattr(
        recommendation_server,
        "product_catalog_stub",
        FakeCatalogStub(catalog_ids),
        raising=False,
    )
    return recommendation_server.RecommendationService()


def test_health_check_returns_serving():
    service = recommendation_server.RecommendationService()
    response = service.Check(None, None)
    assert response.status == health_pb2.HealthCheckResponse.SERVING


def test_recommendations_exclude_products_already_requested(monkeypatch):
    service = make_service(monkeypatch, ["A", "B", "C", "D"])
    request = demo_pb2.ListRecommendationsRequest(product_ids=["A", "B"])
    response = service.ListRecommendations(request, None)
    assert set(response.product_ids) == {"C", "D"}


def test_recommendations_return_at_most_five(monkeypatch):
    catalog = [f"P{i}" for i in range(10)]
    service = make_service(monkeypatch, catalog)
    request = demo_pb2.ListRecommendationsRequest(product_ids=[])
    response = service.ListRecommendations(request, None)
    assert len(response.product_ids) == 5
    assert len(set(response.product_ids)) == 5