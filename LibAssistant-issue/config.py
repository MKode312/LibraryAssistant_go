import os

# Service configuration via environment variables
HOST = os.environ.get('ISSUE_HOST', '0.0.0.0')
PORT = int(os.environ.get('ISSUE_PORT', '50051'))
DB_PATH = os.environ.get('ISSUE_DB_PATH', 'issue_service.db')
CACHE_TTL = int(os.environ.get('ISSUE_CACHE_TTL', '60'))
FINE_PER_DAY = int(os.environ.get('ISSUE_FINE_PER_DAY', '1'))
LOST_FINE = int(os.environ.get('ISSUE_LOST_FINE', '100'))
