"""Orders service for Online Boutique.

Receives finished orders from checkoutservice and stores them in a SQL
database (PostgreSQL on RDS in production, SQLite for tests).
"""
import json
import os

from flask import Flask, jsonify, request
from prometheus_flask_exporter import PrometheusMetrics
from sqlalchemy import Column, DateTime, Float, String, Text, create_engine, func, select
from sqlalchemy.orm import declarative_base, sessionmaker

Base = declarative_base()


class Order(Base):
    __tablename__ = "orders"
    order_id = Column(String(64), primary_key=True)
    user_id = Column(String(64))
    email = Column(String(255))
    tracking_id = Column(String(64))
    currency = Column(String(8))
    total = Column(Float)
    items = Column(Text)  # JSON list of {product_id, quantity}
    created_at = Column(DateTime, server_default=func.now())

    def to_dict(self):
        return {
            "order_id": self.order_id,
            "user_id": self.user_id,
            "email": self.email,
            "tracking_id": self.tracking_id,
            "currency": self.currency,
            "total": self.total,
            "items": json.loads(self.items or "[]"),
            "created_at": self.created_at.isoformat() if self.created_at else None,
        }


def create_app(database_url=None):
    app = Flask(__name__)
    PrometheusMetrics(app)
    engine = create_engine(database_url or os.environ["DATABASE_URL"], pool_pre_ping=True)
    Base.metadata.create_all(engine)
    Session = sessionmaker(bind=engine)

    @app.get("/health")
    def health():
        with Session() as s:
            s.execute(select(1))
        return {"status": "ok", "service": "ordersservice"}

    @app.post("/api/orders")
    def create_order():
        d = request.get_json(silent=True) or {}
        if not d.get("order_id") or not isinstance(d.get("items"), list):
            return jsonify(error="order_id and items are required"), 400
        with Session() as s:
            if s.get(Order, d["order_id"]):
                return jsonify(error="order already exists"), 409
            s.add(Order(
                order_id=d["order_id"], user_id=d.get("user_id"), email=d.get("email"),
                tracking_id=d.get("tracking_id"), currency=d.get("currency", "USD"),
                total=d.get("total", 0), items=json.dumps(d["items"]),
            ))
            s.commit()
        return jsonify(order_id=d["order_id"]), 201

    @app.get("/api/orders")
    def list_orders():
        user = request.args.get("user_id")
        with Session() as s:
            q = select(Order).order_by(Order.created_at.desc()).limit(100)
            if user:
                q = q.where(Order.user_id == user)
            return jsonify([o.to_dict() for o in s.scalars(q)])

    @app.get("/api/orders/<order_id>")
    def get_order(order_id):
        with Session() as s:
            o = s.get(Order, order_id)
            return (jsonify(o.to_dict()), 200) if o else (jsonify(error="not found"), 404)

    return app


def app_factory():
    return create_app()
