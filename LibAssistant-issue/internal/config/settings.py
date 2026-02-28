import os

HOST = os.environ.get('ISSUE_HOST', '0.0.0.0')
PORT = int(os.environ.get('ISSUE_PORT', '50051'))
DB_PATH = os.environ.get('ISSUE_DB_PATH', 'issue_service.db')
CACHE_TTL = int(os.environ.get('ISSUE_CACHE_TTL', '60'))
FINE_PER_DAY = int(os.environ.get('ISSUE_FINE_PER_DAY', '1'))
LOST_FINE = int(os.environ.get('ISSUE_LOST_FINE', '100'))

BOOKS_SERVICE_HOST = os.environ.get('BOOKS_SERVICE_HOST', 'books')
BOOKS_SERVICE_PORT = int(os.environ.get('BOOKS_SERVICE_PORT', '40004'))
STUDENTS_SERVICE_HOST = os.environ.get('STUDENTS_SERVICE_HOST', 'students')
STUDENTS_SERVICE_PORT = int(os.environ.get('STUDENTS_SERVICE_PORT', '50052'))

MAX_DEBTORS_CACHE_SIZE = int(os.environ.get('MAX_DEBTORS_CACHE_SIZE', '1000'))
DEFAULT_ISSUE_DAYS = int(os.environ.get('DEFAULT_ISSUE_DAYS', '14'))
