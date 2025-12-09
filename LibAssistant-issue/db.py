import os
from datetime import datetime, timedelta
from sqlalchemy import (create_engine, Column, Integer, String, DateTime, Boolean, ForeignKey)
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker, relationship

BASE_DIR = os.path.dirname(__file__)
DB_PATH = os.path.join(BASE_DIR, 'issue_service.db')
ENGINE = create_engine(f'sqlite:///{DB_PATH}', connect_args={"check_same_thread": False})
SessionLocal = sessionmaker(bind=ENGINE)
Base = declarative_base()


class Book(Base):
    __tablename__ = 'books'
    id = Column(Integer, primary_key=True)
    title = Column(String, nullable=False)
    total_copies = Column(Integer, default=1)
    available_copies = Column(Integer, default=1)


class Student(Base):
    __tablename__ = 'students'
    id = Column(Integer, primary_key=True)
    name = Column(String, nullable=False)


class Issue(Base):
    __tablename__ = 'issues'
    id = Column(Integer, primary_key=True)
    book_id = Column(Integer, ForeignKey('books.id'), nullable=False)
    student_id = Column(Integer, ForeignKey('students.id'), nullable=False)
    issue_date = Column(DateTime, default=datetime.utcnow)
    due_date = Column(DateTime)
    return_date = Column(DateTime, nullable=True)
    lost = Column(Boolean, default=False)
    overdue_days = Column(Integer, default=0)
    fine = Column(Integer, default=0)

    book = relationship('Book')
    student = relationship('Student')


def init_db(seed=True):
    Base.metadata.create_all(ENGINE)
    if seed:
        session = SessionLocal()
        try:
            if session.query(Book).count() == 0:
                b1 = Book(title='Математика 7 класс', total_copies=3, available_copies=3)
                b2 = Book(title='Русский язык 8 класс', total_copies=2, available_copies=2)
                session.add_all([b1, b2])
            if session.query(Student).count() == 0:
                s1 = Student(name='Иванов Иван')
                s2 = Student(name='Петров Петр')
                session.add_all([s1, s2])
            session.commit()
        finally:
            session.close()


def calculate_overdue_and_fine(issue: Issue):
    now = datetime.utcnow()
    if issue.return_date:
        reference = issue.return_date
    else:
        reference = now
    if issue.due_date and reference > issue.due_date and not issue.lost:
        delta = reference - issue.due_date
        days = delta.days
        issue.overdue_days = days
        issue.fine = days * 1  # 1 unit per day
    elif issue.lost:
        issue.overdue_days = 0
        issue.fine = 100  # flat lost fine
    else:
        issue.overdue_days = 0
        issue.fine = 0


def update_all_overdues(session):
    issues = session.query(Issue).filter(Issue.return_date == None).all()
    for it in issues:
        calculate_overdue_and_fine(it)
    session.commit()
