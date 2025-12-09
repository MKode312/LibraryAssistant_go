import os
import json
from datetime import datetime, timedelta

BASE_DIR = os.path.dirname(__file__)
CACHE_DIR = os.path.join(BASE_DIR, 'cache')
os.makedirs(CACHE_DIR, exist_ok=True)
DEBTORS_CACHE = os.path.join(CACHE_DIR, 'debtors_cache.json')
DEFAULT_TTL = 60  # seconds


def load_debtors_cache():
    try:
        with open(DEBTORS_CACHE, 'r', encoding='utf-8') as f:
            data = json.load(f)
        ts = datetime.fromisoformat(data.get('ts'))
        ttl = data.get('ttl', DEFAULT_TTL)
        if datetime.utcnow() - ts < timedelta(seconds=ttl):
            return data.get('debts')
        return None
    except Exception:
        return None


def save_debtors_cache(debts, ttl=DEFAULT_TTL):
    data = {
        'ts': datetime.utcnow().isoformat(),
        'ttl': ttl,
        'debts': debts,
    }
    with open(DEBTORS_CACHE, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False, default=str)
