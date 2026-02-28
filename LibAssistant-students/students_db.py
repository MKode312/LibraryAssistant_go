import sqlite3
import os

DB_PATH = os.path.dirname(__file__)

def get_connection():
    return sqlite3.connect(f'{DB_PATH}/students_service.db')

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