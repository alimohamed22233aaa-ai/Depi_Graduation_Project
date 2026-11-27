import demo_pb2
from grpc_health.v1 import health_pb2

import email_server


def test_health_check_returns_serving():
    service = email_server.DummyEmailService()
    response = service.Check(None, None)
    assert response.status == health_pb2.HealthCheckResponse.SERVING


def test_send_order_confirmation_returns_empty():
    service = email_server.DummyEmailService()
    request = demo_pb2.SendOrderConfirmationRequest(
        email="customer@example.com",
        order=demo_pb2.OrderResult(order_id="ORD-1"),
    )
    response = service.SendOrderConfirmation(request, None)
    assert isinstance(response, demo_pb2.Empty)


def test_confirmation_template_contains_order_details():
    order = demo_pb2.OrderResult(
        order_id="ORD-123",
        shipping_tracking_id="TRACK-456",
    )
    html = email_server.template.render(order=order)
    assert "ORD-123" in html
    assert "TRACK-456" in html