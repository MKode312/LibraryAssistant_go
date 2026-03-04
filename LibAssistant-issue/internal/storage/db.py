import os
from datetime import datetime

from sqlalchemy import Boolean, Column, DateTime, ForeignKey, Integer, String, create_engine, text
from sqlalchemy.orm import declarative_base, relationship, sessionmaker

STORAGE_PATH = os.path.join('.', 'issue_service.db')  # создастся в рабочей директории
ENGINE = create_engine(f'sqlite:///{STORAGE_PATH}', connect_args={'check_same_thread': False})
SessionLocal = sessionmaker(bind=ENGINE)
Base = declarative_base()

class Book(Base):
    __tablename__ = 'books'

    id = Column(String, primary_key=True)
    title = Column(String, nullable=False)
    total_copies = Column(Integer, default=1)
    available_copies = Column(Integer, default=1)


class Student(Base):
    __tablename__ = 'students'

    id = Column(String, primary_key=True)
    name = Column(String, nullable=False)


class Issue(Base):
    __tablename__ = 'issues'

    id = Column(String, primary_key=True)
    book_id = Column(String, ForeignKey('books.id'), nullable=False)
    student_id = Column(String, ForeignKey('students.id'), nullable=False)
    issue_date = Column(DateTime, default=datetime.utcnow)
    due_date = Column(DateTime)
    return_date = Column(DateTime, nullable=True)
    lost = Column(Boolean, default=False)
    overdue_days = Column(Integer, default=0)
    fine = Column(Integer, default=0)

    book = relationship('Book')
    student = relationship('Student')


def init_db():
    Base.metadata.create_all(ENGINE)
    with ENGINE.begin() as conn:
        conn.execute(text(
            "CREATE UNIQUE INDEX IF NOT EXISTS uq_active_issue_student_book "
            "ON issues(book_id, student_id) WHERE return_date IS NULL"
        ))


def calculate_overdue_and_fine(issue: Issue):
    reference = issue.return_date or datetime.utcnow()
    if issue.lost:
        issue.overdue_days = 0
        issue.fine = 100
        return

    if issue.due_date and reference > issue.due_date:
        days = (reference - issue.due_date).days
        issue.overdue_days = days
        issue.fine = days
        return

    issue.overdue_days = 0
    issue.fine = 0


def update_all_overdues(session):
    issues = session.query(Issue).filter(Issue.return_date == None).all()
    for issue in issues:
        calculate_overdue_and_fine(issue)
    session.commit()
