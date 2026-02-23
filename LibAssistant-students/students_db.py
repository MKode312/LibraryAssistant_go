import sqlite3
import os

DB_PATH = "/app/storage"

def get_connection():
    os.makedirs(DB_PATH, exist_ok=True)
    return sqlite3.connect(os.path.join(DB_PATH, 'students_service.db'))

def init_db():
    with get_connection() as con:
        cur = con.cursor()
        cur.execute("""CREATE TABLE IF NOT EXISTS students (
                id TEXT NOT NULL UNIQUE,
                full_name TEXT NOT NULL UNIQUE,
                grade INTEGER NOT NULL,
                letter TEXT NOT NULL
                )""")
        con.commit()