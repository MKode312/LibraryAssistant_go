import json
import logging
from datetime import datetime
from typing import Dict, Optional

logger = logging.getLogger(__name__)
_producer = None


def init_kafka_producer(bootstrap_servers: Optional[list] = None, enabled: bool = True):
    global _producer
    if not enabled:
        _producer = None
        return

    try:
        from kafka import KafkaProducer  # type: ignore

        servers = bootstrap_servers or ['kafka:9092']
        _producer = KafkaProducer(
            bootstrap_servers=servers,
            value_serializer=lambda value: json.dumps(value, ensure_ascii=False).encode('utf-8'),
        )
        logger.info('Kafka producer initialized: %s', servers)
    except Exception as exc:
        logger.warning('Kafka disabled: %s', exc)
        _producer = None


def publish_event(event_type: str, data: Dict):
    global _producer

    payload = dict(data)
    payload.setdefault('timestamp', datetime.utcnow().isoformat())

    if _producer is None:
        logger.info('[KAFKA_DISABLED] %s: %s', event_type, json.dumps(payload, ensure_ascii=False))
        return

    try:
        _producer.send(event_type, value=payload)
    except Exception as exc:
        logger.error('Kafka publish failed: %s', exc, exc_info=True)


def close_kafka_producer():
    global _producer
    if _producer is None:
        return
    try:
        _producer.close()
    finally:
        _producer = None
